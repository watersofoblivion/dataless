package component

import (
	"encoding/json"
	"log"

	"github.com/oliveagle/jsonpath"
	"github.com/qri-io/jsonschema"
)

type Endpoint struct {
	// Map of parameter names to their types
	Parameters map[string]*Parameter `yaml:"Parameters,omitempty"`

	// Schema of the request body
	Body *jsonschema.Schema `yaml:"Body,omitempty"`

	// The method and URI to make the request against
	Method string `yaml:"Method"`
	URI    string `yaml:"URI"`

	// Map of status codes to their response definitions
	Responses map[string]*Response `yaml:"Responses"`
}

type Parameter struct {
	Type     string             `yaml:"Type"`
	In       string             `yaml:"In"`
	Required bool               `yaml:"Required,omitempty"`
	Schema   *jsonschema.Schema `yaml:"Schema,omitempty"`
}

func (parameter *Parameter) UnmarshalYAML(unmarshal func(v interface{}) error) (err error) {
	v := make(map[string]interface{})
	if err := unmarshal(&v); err != nil {
		return err
	}

	parameter.Type = v["Type"].(string)
	parameter.In = v["In"].(string)
	parameter.Required = v["Required"].(bool)

	if schema, found := v["Schema"]; found {
		bs, err := json.Marshal(schema)
		if err != nil {
			return err
		}

		parameter.Schema = &jsonschema.Schema{}
		if err := json.Unmarshal(bs, &parameter.Schema); err != nil {
			return err
		}
	}

	return nil
}

type Response struct {
	Outputs map[string]*Output `yaml:"Outputs,omitempty"`
	Schema  *jsonschema.Schema `yaml:"Schema,omitempty"`
}

func (response *Response) UnmarshalYAML(unmarshal func(v interface{}) error) (err error) {
	v := make(map[string]interface{})
	if err := unmarshal(&v); err != nil {
		return err
	}

	response.Outputs = v["Type"].(map[string]*Output)

	if schema, found := v["Schema"]; found {
		bs, err := json.Marshal(schema)
		if err != nil {
			return err
		}

		response.Schema = &jsonschema.Schema{}
		if err := json.Unmarshal(bs, &response.Schema); err != nil {
			return err
		}
	}

	return nil
}

type OutputType string

const (
	OutputTypeHeader OutputType = "Header"
	OutputTypeBody   OutputType = "Body"
)

type Output struct {
	Type OutputType `yaml:"Type"`
	// A JSONPath to the value in the response
	Location string `yaml:"Path"`
	jsonPath *jsonpath.Compiled
}

func (output *Output) UnmarshalYAML(unmarshal func(v interface{}) error) (err error) {
	v := make(map[string]interface{})
	if err := unmarshal(&v); err != nil {
		return err
	}

	output.Location = v["Path"].(string)
	output.Type = OutputType(v["Type"].(string))
	switch output.Type {
	case OutputTypeBody:
		output.jsonPath, err = jsonpath.Compile(output.Location)
		if err != nil {
			return err
		}
	default:
		log.Panicf("unknown output type %q", output.Type)
	}

	return nil
}
