package env

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/hashicorp/nomad/nomad/mock"
	"github.com/hashicorp/nomad/nomad/structs"
)

const (
	// Node values that tests can rely on
	metaKey   = "instance"
	metaVal   = "t2-micro"
	attrKey   = "arch"
	attrVal   = "amd64"
	nodeName  = "test node"
	nodeClass = "test class"

	// Environment variable values that tests can rely on
	envOneKey = "NOMAD_IP"
	envOneVal = "127.0.0.1"
	envTwoKey = "NOMAD_PORT_WEB"
	envTwoVal = ":80"
)

var (
	// Networks that tests can rely on
	networks = []*structs.NetworkResource{
		&structs.NetworkResource{
			IP:            "127.0.0.1",
			ReservedPorts: []structs.Port{{"http", 80}},
			DynamicPorts:  []structs.Port{{"https", 8080}},
		},
	}
	portMap = map[string]int{
		"https": 443,
	}
)

func testTaskEnvironment() *TaskEnvironment {
	n := mock.Node()
	n.Attributes = map[string]string{
		attrKey: attrVal,
	}
	n.Meta = map[string]string{
		metaKey: metaVal,
	}
	n.Name = nodeName
	n.NodeClass = nodeClass

	envVars := map[string]string{
		envOneKey: envOneVal,
		envTwoKey: envTwoVal,
	}
	return NewTaskEnvironment(n).SetEnvvars(envVars).Build()
}

func TestEnvironment_ParseAndReplace_Env(t *testing.T) {
	env := testTaskEnvironment()

	input := []string{fmt.Sprintf(`"${%v}"!`, envOneKey), fmt.Sprintf("${%s}${%s}", envOneKey, envTwoKey)}
	act := env.ParseAndReplace(input)
	exp := []string{fmt.Sprintf(`"%s"!`, envOneVal), fmt.Sprintf("%s%s", envOneVal, envTwoVal)}

	if !reflect.DeepEqual(act, exp) {
		t.Fatalf("ParseAndReplace(%v) returned %#v; want %#v", input, act, exp)
	}
}

func TestEnvironment_ParseAndReplace_Meta(t *testing.T) {
	input := []string{fmt.Sprintf("${%v%v}", nodeMetaPrefix, metaKey)}
	exp := []string{metaVal}
	env := testTaskEnvironment()
	act := env.ParseAndReplace(input)

	if !reflect.DeepEqual(act, exp) {
		t.Fatalf("ParseAndReplace(%v) returned %#v; want %#v", input, act, exp)
	}
}

func TestEnvironment_ParseAndReplace_Attr(t *testing.T) {
	input := []string{fmt.Sprintf("${%v%v}", nodeAttributePrefix, attrKey)}
	exp := []string{attrVal}
	env := testTaskEnvironment()
	act := env.ParseAndReplace(input)

	if !reflect.DeepEqual(act, exp) {
		t.Fatalf("ParseAndReplace(%v) returned %#v; want %#v", input, act, exp)
	}
}

func TestEnvironment_ParseAndReplace_Node(t *testing.T) {
	input := []string{fmt.Sprintf("${%v}", nodeNameKey), fmt.Sprintf("${%v}", nodeClassKey)}
	exp := []string{nodeName, nodeClass}
	env := testTaskEnvironment()
	act := env.ParseAndReplace(input)

	if !reflect.DeepEqual(act, exp) {
		t.Fatalf("ParseAndReplace(%v) returned %#v; want %#v", input, act, exp)
	}
}

func TestEnvironment_ParseAndReplace_Mixed(t *testing.T) {
	input := []string{
		fmt.Sprintf("${%v}${%v%v}", nodeNameKey, nodeAttributePrefix, attrKey),
		fmt.Sprintf("${%v}${%v%v}", nodeClassKey, nodeMetaPrefix, metaKey),
		fmt.Sprintf("${%v}${%v}", envTwoKey, nodeClassKey),
	}
	exp := []string{
		fmt.Sprintf("%v%v", nodeName, attrVal),
		fmt.Sprintf("%v%v", nodeClass, metaVal),
		fmt.Sprintf("%v%v", envTwoVal, nodeClass),
	}
	env := testTaskEnvironment()
	act := env.ParseAndReplace(input)

	if !reflect.DeepEqual(act, exp) {
		t.Fatalf("ParseAndReplace(%v) returned %#v; want %#v", input, act, exp)
	}
}

func TestEnvironment_ReplaceEnv_Mixed(t *testing.T) {
	input := fmt.Sprintf("${%v}${%v%v}", nodeNameKey, nodeAttributePrefix, attrKey)
	exp := fmt.Sprintf("%v%v", nodeName, attrVal)
	env := testTaskEnvironment()
	act := env.ReplaceEnv(input)

	if act != exp {
		t.Fatalf("ParseAndReplace(%v) returned %#v; want %#v", input, act, exp)
	}
}

func TestEnvironment_AsList(t *testing.T) {
	n := mock.Node()
	env := NewTaskEnvironment(n).
		SetNetworks(networks).
		SetPortMap(portMap).
		SetMeta(map[string]string{"foo": "baz"}).Build()

	act := env.EnvList()
	exp := []string{
		"NOMAD_ADDR_http=127.0.0.1:80",
		"NOMAD_ADDR_https=127.0.0.1:443",
		"NOMAD_HOST_PORT_https=443",
		"NOMAD_META_FOO=baz",
	}
	sort.Strings(act)
	sort.Strings(exp)
	if !reflect.DeepEqual(act, exp) {
		t.Fatalf("env.List() returned %v; want %v", act, exp)
	}
}

func TestEnvironment_ClearEnvvars(t *testing.T) {
	n := mock.Node()
	env := NewTaskEnvironment(n).
		SetNetworks(networks).
		SetPortMap(portMap).
		SetEnvvars(map[string]string{"foo": "baz", "bar": "bang"}).Build()

	act := env.EnvList()
	exp := []string{
		"NOMAD_ADDR_http=127.0.0.1:80",
		"NOMAD_ADDR_https=127.0.0.1:443",
		"NOMAD_HOST_PORT_https=443",
		"bar=bang",
		"foo=baz",
	}
	sort.Strings(act)
	sort.Strings(exp)
	if !reflect.DeepEqual(act, exp) {
		t.Fatalf("env.List() returned %v; want %v", act, exp)
	}

	// Clear the environent variables.
	env.ClearEnvvars().Build()

	act = env.EnvList()
	exp = []string{
		"NOMAD_ADDR_http=127.0.0.1:80",
		"NOMAD_ADDR_https=127.0.0.1:443",
		"NOMAD_HOST_PORT_https=443",
	}
	sort.Strings(act)
	sort.Strings(exp)
	if !reflect.DeepEqual(act, exp) {
		t.Fatalf("env.List() returned %v; want %v", act, exp)
	}
}

func TestEnvironment_Interprolate(t *testing.T) {
	env := testTaskEnvironment().
		SetEnvvars(map[string]string{"test": "${node.class}", "test2": "${attr.arch}"}).
		Build()

	act := env.EnvList()
	exp := []string{fmt.Sprintf("test=%s", nodeClass), fmt.Sprintf("test2=%s", attrVal)}
	sort.Strings(act)
	sort.Strings(exp)
	if !reflect.DeepEqual(act, exp) {
		t.Fatalf("env.List() returned %v; want %v", act, exp)
	}
}
