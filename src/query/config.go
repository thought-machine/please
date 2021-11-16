package query

import (
	"bytes"
	"encoding/json"
	"os"
	"fmt"

	"github.com/please-build/gcfg"
	"github.com/thought-machine/please/src/core"
)

// Config prints the configuration settings as JSON.
func Config(config *core.Configuration, option string, printJson bool) {
	if printJson {
		v, err := gcfg.RawJSON(config)
		if err != nil {
			log.Fatal(err)
		}

		var out bytes.Buffer
		if err := json.Indent(&out, v, "", "    "); err != nil {
			fmt.Println("error:", err)
		}
		os.Stdout.Write(out.Bytes())
	}

	v, err := gcfg.Stringify(config)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", v)

	fmt.Println()
	fmt.Println()
	

	//t, err := gcfg.Get(config, option)
	//if err != nil {
		//log.Fatal(err)
	//}
	//fmt.Printf("%s\n", t)
}

//func printConfigAsJSON(config *core.Configuration, option string) {
	//encoder := json.NewEncoder(os.Stdout)
	//encoder.SetIndent("", "    ")
	//encoder.SetEscapeHTML(false)

	//var value interface{} = config
	//if option != "" {
		//reflectValue, err := config.RetrieveOption(option)
		//if err != nil {
			//log.Fatal(err)
		//}
		//value = reflectValue.Interface()
	//}

	//if err := encoder.Encode(value); err != nil {
		//log.Fatalf("Failed to serialise configuration as JSON: %s", err)
	//}
//}
