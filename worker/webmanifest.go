package worker

type WebAppManifest struct {
	BackgroundColor           string               `json:"background_color"`
	Categories                []string             `json:"categories"`
	Description               string               `json:"description"`
	Display                   string               `json:"display"`
	DisplayOverride           string               `json:"display_override"`
	FileHandlers              []FileHandler        `json:"file_handlers"`
	Icons                     []Icon               `json:"icons"`
	ID                        string               `json:"id"`
	LaunchHandler             LaunchHandler        `json:"launch_handler"`
	Name                      string               `json:"name"`
	Orientation               string               `json:"orientation"`
	PreferRelatedApplications bool                 `json:"prefer_related_applications"`
	ProtocolHandlers          []ProtocolHandler    `json:"protocol_handlers"`
	RelatedApplications       []RelatedApplication `json:"related_applications"`
	Scope                     string               `json:"scope"`
	Screenshots               []Screenshot         `json:"screenshots"`
	ServiceWorker             ServiceWorker        `json:"service_worker"`
	ShareTarget               ShareTarget          `json:"share_target"`
	Shortcuts                 []Shortcut           `json:"shortcuts"`
	ShortName                 string               `json:"short_name"`
	StartURL                  string               `json:"start_url"`
	ThemeColor                string               `json:"theme_color"`
}

type Shortcut struct {
	Name        string `json:"name"`
	ShortName   string `json:"short_name"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Icons       []Icon `json:"icons"`
}

type ServiceWorker struct {
	Scope    string `json:"scope"`
	Src      string `json:"src"`
	UseCache bool   `json:"use_cache"`
}

type ShareTarget struct {
	EncType string `json:"enctype"`
	Method  string `json:"method"`
	Params  struct {
		Title string `json:"title"`
		Text  string `json:"text"`
		URL   string `json:"url"`
		Files []struct {
			Name   string   `json:"name"`
			Accept []string `json:"accept"`
		} `json:"files"`
	} `json:"params"`
}

type Screenshot struct {
	Src        string `json:"src"`
	Type       string `json:"type"`
	Sizes      string `json:"sizes"`
	FormFactor string `json:"form_factor"`
	Label      string `json:"label"`
}

type RelatedApplication struct {
	Platform string `json:"platform"`
	Url      string `json:"url"`
	ID       string `json:"id"`
}

type ProtocolHandler struct {
	Protocol string `json:"protocol"`
	Url      string `json:"url"`
}

type LaunchHandler struct {
	ClientMode string `json:"client_mode"`
}

type Icon struct {
	Src   string `json:"src"`
	Sizes string `json:"sizes"`
	Type  string `json:"type"`
}

type FileHandler struct {
	Action string              `json:"action"`
	Accept map[string][]string `json:"accept"`
}
