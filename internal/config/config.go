package config

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/mitchellh/mapstructure"
)

// PermissionConfig represents a permission with allow/deny rules
type PermissionConfig struct {
	AllowAll bool
	DenyAll  bool
	Allow    []string
	Deny     []string
}

// PermissionSet represents the collection of permissions
type PermissionSet struct {
	All    bool             `mapstructure:"all"`
	Read   PermissionConfig `mapstructure:"read"`
	Write  PermissionConfig `mapstructure:"write"`
	Import PermissionConfig `mapstructure:"import"`
	Env    PermissionConfig `mapstructure:"env"`
	Net    PermissionConfig `mapstructure:"net"`
	Run    PermissionConfig `mapstructure:"run"`
	Ffi    PermissionConfig `mapstructure:"ffi"`
	Sys    PermissionConfig `mapstructure:"sys"`
}

// Flags converts a PermissionConfig to Deno CLI flags
func (p *PermissionConfig) Flags(flagName string) []string {
	var flags []string

	if p.AllowAll {
		flags = append(flags, fmt.Sprintf("--allow-%s", flagName))
	} else if p.DenyAll {
		flags = append(flags, fmt.Sprintf("--deny-%s", flagName))
	} else {
		if len(p.Allow) > 0 {
			flags = append(flags, fmt.Sprintf("--allow-%s=%s", flagName, strings.Join(p.Allow, ",")))
		}
		if len(p.Deny) > 0 {
			flags = append(flags, fmt.Sprintf("--deny-%s=%s", flagName, strings.Join(p.Deny, ",")))
		}
	}

	return flags
}

// Flags converts a PermissionSet to Deno CLI flags
func (ps *PermissionSet) Flags() []string {
	var flags []string

	if ps.All {
		flags = append(flags, "--allow-all")
		return flags
	}

	flags = append(flags, ps.Read.Flags("read")...)
	flags = append(flags, ps.Write.Flags("write")...)
	flags = append(flags, ps.Import.Flags("import")...)
	flags = append(flags, ps.Env.Flags("env")...)
	flags = append(flags, ps.Net.Flags("net")...)
	flags = append(flags, ps.Run.Flags("run")...)
	flags = append(flags, ps.Ffi.Flags("ffi")...)
	flags = append(flags, ps.Sys.Flags("sys")...)

	return flags
}

// PermissionConfigDecodeHook is a mapstructure decode hook for PermissionConfig
func PermissionConfigDecodeHook() mapstructure.DecodeHookFunc {
	return func(from reflect.Type, to reflect.Type, data interface{}) (interface{}, error) {
		if to != reflect.TypeOf(PermissionConfig{}) {
			return data, nil
		}

		var config PermissionConfig

		// Handle boolean
		if from.Kind() == reflect.Bool {
			if data.(bool) {
				config.AllowAll = true
			} else {
				config.DenyAll = true
			}
			return config, nil
		}

		// Handle array of strings
		if from.Kind() == reflect.Slice {
			if slice, ok := data.([]interface{}); ok {
				for _, item := range slice {
					if str, ok := item.(string); ok {
						config.Allow = append(config.Allow, str)
					}
				}
				return config, nil
			}
		}

		// Handle map with allow/deny keys
		if from.Kind() == reflect.Map {
			dataMap, ok := data.(map[string]interface{})
			if !ok {
				return data, nil
			}

			if allow, hasAllow := dataMap["allow"]; hasAllow {
				switch v := allow.(type) {
				case bool:
					if v {
						config.AllowAll = true
					} else {
						config.DenyAll = true
					}
				case []interface{}:
					for _, item := range v {
						if str, ok := item.(string); ok {
							config.Allow = append(config.Allow, str)
						}
					}
				}
			}

			if deny, hasDeny := dataMap["deny"]; hasDeny {
				switch v := deny.(type) {
				case bool:
					if v {
						config.DenyAll = true
					} else {
						config.AllowAll = true
					}
				case []interface{}:
					for _, item := range v {
						if str, ok := item.(string); ok {
							config.Deny = append(config.Deny, str)
						}
					}
				}
			}

			return config, nil
		}

		return data, nil
	}
}

// DecodePermissionSet decodes a map into a PermissionSet
func DecodePermissionSet(input map[string]interface{}) (*PermissionSet, error) {
	var result PermissionSet

	config := &mapstructure.DecoderConfig{
		DecodeHook: PermissionConfigDecodeHook(),
		Result:     &result,
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return nil, err
	}

	if err := decoder.Decode(input); err != nil {
		return nil, err
	}

	return &result, nil
}

// Config represents the top-level configuration
type Config struct {
	Domain            string                   `mapstructure:"domain"`
	AdditionalDomains []string                 `mapstructure:"additionalDomains"`
	AuthorizedKeys    []string                 `mapstructure:"authorizedKeys"`
	Permissions       map[string]PermissionSet `mapstructure:"permissions"`
}

// MergePermissionConfig merges two PermissionConfig objects, with the second taking precedence
func MergePermissionConfig(base, override PermissionConfig) PermissionConfig {
	result := base

	// If override has AllowAll or DenyAll set, it takes complete precedence
	if override.AllowAll {
		return PermissionConfig{AllowAll: true}
	}

	if override.DenyAll {
		return PermissionConfig{DenyAll: true}
	}

	if base.AllowAll {
		return PermissionConfig{AllowAll: true}
	}

	if base.DenyAll {
		return PermissionConfig{DenyAll: true}
	}

	// Otherwise merge the lists
	if len(override.Allow) > 0 {
		result.Allow = append(result.Allow, override.Allow...)
	}

	if len(override.Deny) > 0 {
		result.Deny = append(result.Deny, override.Deny...)
	}

	return result
}

// MergePermissionSets merges multiple PermissionSet objects from left to right
// Later sets take precedence over earlier ones
func MergePermissionSets(sets ...*PermissionSet) *PermissionSet {
	if len(sets) == 0 {
		return &PermissionSet{}
	}

	result := &PermissionSet{}

	for _, set := range sets {
		if set == nil {
			continue
		}

		// If any set has All=true, it takes complete precedence
		if set.All {
			result.All = true
			return result
		}

		result.Read = MergePermissionConfig(result.Read, set.Read)
		result.Write = MergePermissionConfig(result.Write, set.Write)
		result.Import = MergePermissionConfig(result.Import, set.Import)
		result.Env = MergePermissionConfig(result.Env, set.Env)
		result.Net = MergePermissionConfig(result.Net, set.Net)
		result.Run = MergePermissionConfig(result.Run, set.Run)
		result.Ffi = MergePermissionConfig(result.Ffi, set.Ffi)
		result.Sys = MergePermissionConfig(result.Sys, set.Sys)
	}

	return result
}
