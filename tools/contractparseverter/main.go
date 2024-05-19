package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"

	ei "biehdc.reegg/eggpb"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func panicIfErr(err error) {
	if err != nil {
		panic(err)
	}
}

// convert cerns contracts.json into a golang and plaintext contracts_go.json
func main() {
	// open the file
	ogcontractsfile, err := os.Open("contracts.json")
	panicIfErr(err)
	defer ogcontractsfile.Close()

	// read the whole file
	datacontent, err := io.ReadAll(ogcontractsfile)
	panicIfErr(err)

	type upstream struct {
		Id    string `json:"id"`
		Proto string `json:"proto"`
	}

	// parse all the entries of the og file into an array
	var data []upstream
	err = json.Unmarshal(datacontent, &data)
	panicIfErr(err)

	// decode the protobuf messages into proper structs
	var asprotobuf []*ei.Contract
	for _, contract := range data {
		cnt, err := base64.StdEncoding.DecodeString(contract.Proto)
		panicIfErr(err)

		var protod ei.Contract
		err = proto.Unmarshal(cnt, &protod)
		panicIfErr(err)
		//fmt.Println(protod.String())
		asprotobuf = append(asprotobuf, &protod)
	}

	// assemble the json with the proper protobuf messages
	// it needs its own library because stdlib json cant
	// do it correctly (accoding to protobuf docs)
	var jsonobjects bytes.Buffer
	jsonobjects.WriteString("[")
	for _, cnt := range asprotobuf {
		marshalled, err := protojson.Marshal(cnt)
		panicIfErr(err)
		//fmt.Println(string(bytes))
		jsonobjects.WriteString(string(marshalled))
		jsonobjects.WriteString(",")
	}
	// trim the last comma because ignoring it is too hard
	jsonobjects.Truncate(jsonobjects.Len() - 1)
	jsonobjects.WriteString("]")

	// finally use stdlib json to make it look pretty
	var prettyJSON bytes.Buffer
	err = json.Indent(&prettyJSON, jsonobjects.Bytes(), "", "\t")
	panicIfErr(err)
	fmt.Println(prettyJSON.String())
}
