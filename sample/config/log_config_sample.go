package config

import (
	"encoding/json"
	"fmt"

	"github.com/eluv-io/log-go"
)

func init() {
	// logging is configured with a configuration object.
	config := &log.Config{
		Level:   "trace",
		Handler: "console",
		Named: map[string]*log.Config{
			"/eluvio/log/sample/sub": {
				Level:   "normal",
				Handler: "json",
			},
		},
	}
	log.SetDefault(config)

	// The config can also be specified in json ...
	_, _ = loadConfigJSON()

	buf, _ := json.Marshal(config)
	fmt.Println(string(buf))
}

func loadConfigJSON() (*log.Config, error) {
	txt := `
{
  "level": "debug",
  "formatter": "text",
  "named": {
    "/eluvio/log/sample/sub": {
      "level": "normal",
      "formatter": "json"
    }
  }
}
`
	var c = &log.Config{}
	err := json.Unmarshal([]byte(txt), c)
	return c, err
}
