package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"

	goyaml "gopkg.in/yaml.v2"
	"github.com/ghodss/yaml"
)

func main() {
	yamltojson := flag.Bool("yamltojson", false, "Convert yaml to json instead of the default json to yaml.")
	flag.Parse()

	// Don't wrap long lines
	goyaml.FutureLineWrap()

	inBytes, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}

	var outBytes []byte
	if *yamltojson {
		outBytes, err = yaml.YAMLToJSON(inBytes)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		outBytes, err = yaml.JSONToYAML(inBytes)
		if err != nil {
			log.Fatal(err)
		}
	}

	os.Stdout.Write(outBytes)
}
