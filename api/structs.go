package api

type ConfigStruct struct {
	Host string                  `json:"host"`
	Port string                  `json:"port"`
	APIKeys map[string]APIKeyStruct `json:"apiKeys"`
}

type PermissionsStruct struct {
	Bastille  []string `json:"bastille"`
	Rocinante []string `json:"rocinante"`
	Admin     []string `json:"admin"`
}

type APIKeyStruct struct {
	Salt        string            `json:"salt"`
	Hash        string            `json:"hash"`
	Permissions PermissionsStruct `json:"permissions"`
}

type BastilleSpecStruct struct {
	Software string                  `json:"software"`
	Commands []BastilleCommandStruct `json:"commands"`
}

type BastilleOptionStruct struct {
	SFlag string      `json:"sflag"`
	LFlag string      `json:"lflag"`
	Text  string      `json:"text"`
	Value interface{} `json:"value,omitempty"`
}

type BastilleCommandStruct struct {
	Command     string                 `json:"command"`
	Description string                 `json:"description"`
	Options     []BastilleOptionStruct `json:"options"`
	Parameters  []string               `json:"parameters"`
}

type BastilleCommandOutputStruct struct {
	output string
	port   string
}

type RocinanteSpecStruct struct {
	Software string                   `json:"software"`
	Commands []RocinanteCommandStruct `json:"commands"`
}

type RocinanteOptionStruct struct {
	SFlag string      `json:"sflag"`
	LFlag string      `json:"lflag"`
	Text  string      `json:"text"`
	Value interface{} `json:"value,omitempty"`
}

type RocinanteCommandStruct struct {
	Command    string                  `json:"command"`
	Options    []RocinanteOptionStruct `json:"options"`
	Parameters []string                `json:"parameters"`
}

type RocinanteCommandOutputStruct struct {
	output string
	port   string
}
