package editor

type ProductConfiguration struct {
	NameShort                    string              `json:"nameShort"`
	NameLong                     string              `json:"nameLong"`
	ApplicationName              string              `json:"applicationName"`
	DataFolderName               string              `json:"dataFolderName"`
	Version                      string              `json:"version"`
	ExtensionsGallery            ExtensionsGallery   `json:"extensionsGallery"`
	ExtensionEnabledApiProposals map[string][]string `json:"extensionEnabledApiProposals"`
}

type ExtensionsGallery struct {
	ServiceUrl          string `json:"serviceUrl"`
	ItemUrl             string `json:"itemUrl"`
	ResourceUrlTemplate string `json:"resourceUrlTemplate"`
}

type FolderUri struct {
	Scheme string `json:"scheme"`
	Path   string `json:"path"`
}

type AdditionalBuiltinExtension struct {
	Scheme string `json:"scheme"`
	Path   string `json:"path"`
}

type Configuration struct {
	ProductConfiguration        ProductConfiguration         `json:"productConfiguration"`
	FolderUri                   FolderUri                    `json:"folderUri"`
	AdditionalBuiltinExtensions []AdditionalBuiltinExtension `json:"additionalBuiltinExtensions"`
}

func getProductConfig(rootPath string) *Configuration {
	return &Configuration{
		ProductConfiguration: ProductConfiguration{
			NameShort:       "VSCode Web Sample",
			NameLong:        "VSCode Web without FileSystemProvider",
			ApplicationName: "smallweb-editor",
			DataFolderName:  ".smallweb-editor",
			Version:         "1.91.1",
			ExtensionsGallery: ExtensionsGallery{
				ServiceUrl:          "https://open-vsx.org/vscode/gallery",
				ItemUrl:             "https://open-vsx.org/vscode/item",
				ResourceUrlTemplate: "https://open-vsx.org/vscode/unpkg/{publisher}/{name}/{version}/{path}",
			},
			ExtensionEnabledApiProposals: map[string][]string{
				"pomdtr.smallweb": {"fileSearchProvider", "textSearchProvider"},
			},
		},
		FolderUri: FolderUri{
			Scheme: "smallweb",
			Path:   rootPath,
		},
		AdditionalBuiltinExtensions: []AdditionalBuiltinExtension{
			{
				Scheme: "https",
				Path:   "smallweb",
			},
		},
	}
}
