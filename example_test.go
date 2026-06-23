package vdf_test

import (
	"fmt"
	"strings"

	"github.com/woozymasta/vdf"
)

func ExampleParseString() {
	doc, err := vdf.ParseString(`"root" { "name" "server-1" }`)
	if err != nil {
		fmt.Println(err)
		return
	}

	root := doc.Roots[0]
	fmt.Println(root.Key)
	fmt.Println(*root.First("name").StringValue)

	// Output:
	// root
	// server-1
}

func ExampleParseAuto() {
	doc, err := vdf.ParseAuto([]byte(`"cfg" { "timeout" "5" }`))
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(doc.Format == vdf.FormatText)

	// Output:
	// true
}

func ExampleWriteString() {
	doc := vdf.NewDocumentWithFormat(vdf.FormatText)
	root := vdf.NewObjectNode("app")
	root.Add(vdf.NewStringNode("name", "demo"))
	root.Add(vdf.NewUint32Node("id", 7))
	doc.AddRoot(root)

	text, err := vdf.WriteString(doc)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(strings.Contains(text, `"app"`))
	fmt.Println(strings.Contains(text, `"id"`))

	// Output:
	// true
	// true
}

func ExampleDecoder_WalkEvents() {
	dec := vdf.NewDecoder(strings.NewReader(`"root" { "k" "v" }`), vdf.DecodeOptions{
		Format: vdf.FormatText,
	})

	count := 0
	for {
		_, err := dec.WalkEvents()
		if err != nil {
			break
		}

		count++
	}

	fmt.Println(count)

	// Output:
	// 5
}

func ExampleNewBuilder() {
	doc, err := vdf.NewBuilder("config").
		Set("name", "server").
		SetUint32("port", 2302).
		Object("db", func(b *vdf.Builder) {
			b.Set("host", "localhost")
		}).
		Document()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(doc.Roots[0].Key)
	fmt.Println(*doc.Roots[0].First("name").StringValue)

	// Output:
	// config
	// server
}

func ExampleMarshal() {
	type Server struct {
		Name string `vdf:"name"`
		Port uint32 `vdf:"port"`
	}

	doc, err := vdf.Marshal("Server", Server{Name: "game-1", Port: 2302})
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(doc.Roots[0].Key)
	fmt.Println(*doc.Roots[0].First("name").StringValue)

	// Output:
	// Server
	// game-1
}

func ExampleUnmarshal() {
	type Server struct {
		Name string `vdf:"name"`
		Port uint32 `vdf:"port"`
	}

	doc, _ := vdf.ParseString(`"Server" { "name" "game-1" "port" "2302" }`)

	var s Server
	if err := vdf.Unmarshal(doc, "Server", &s); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(s.Name)
	fmt.Println(s.Port)

	// Output:
	// game-1
	// 2302
}

func ExampleDecoder_NextEvent() {
	dec := vdf.NewDecoder(strings.NewReader(`"root" { "k" "v" }`), vdf.DecodeOptions{
		Format: vdf.FormatText,
	})

	count := 0
	for {
		_, err := dec.NextEvent()
		if err != nil {
			break
		}

		count++
	}

	fmt.Println(count)

	// Output:
	// 5
}
