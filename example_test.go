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
