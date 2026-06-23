package vdf

import (
	"bytes"
	"strings"
	"testing"
)

var (
	// benchTextInput uses a realistic fixture from testdata.
	benchTextInput = mustReadFixtureBytes("consolesample.vdf")
	// benchBinaryDoc is prebuilt once so encode/decode loops measure core paths only.
	benchBinaryDoc = mustBenchDocument()
	// benchBinaryIn is pre-encoded once so binary decode loops avoid setup noise.
	benchBinaryIn = mustBenchBinaryBytes()

	// benchmark sink variables prevent compiler dead-code elimination.
	benchDocSink   *Document
	benchBytesSink []byte
	benchEventSink Event
	benchAnySink   any
)

// mustBenchDocument builds benchmark AST or panics on setup failure.
func mustBenchDocument() *Document {
	doc, err := ParseBytes(benchTextInput, DecodeOptions{Format: FormatText})
	if err != nil {
		panic(err)
	}

	return doc
}

// mustBenchBinaryBytes pre-encodes binary benchmark payload or panics on setup failure.
func mustBenchBinaryBytes() []byte {
	data, err := AppendBinary(nil, benchBinaryDoc, EncodeOptions{Format: FormatBinary})
	if err != nil {
		panic(err)
	}

	return data
}

func BenchmarkReadParseFlow(b *testing.B) {
	b.ReportAllocs()

	b.Run("DecodeTextDocument", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			doc, err := NewDecoder(bytes.NewReader(benchTextInput), DecodeOptions{Format: FormatText}).DecodeDocument()
			if err != nil {
				b.Fatalf("DecodeDocument(text) returned error: %v", err)
			}

			benchDocSink = doc
		}
	})

	b.Run("DecodeBinaryDocument", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			doc, err := NewDecoder(bytes.NewReader(benchBinaryIn), DecodeOptions{Format: FormatBinary}).DecodeDocument()
			if err != nil {
				b.Fatalf("DecodeDocument(binary) returned error: %v", err)
			}

			benchDocSink = doc
		}
	})
}

func BenchmarkWriteFormatFlow(b *testing.B) {
	b.ReportAllocs()

	b.Run("EncodeText", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			out, err := AppendText(nil, benchBinaryDoc, EncodeOptions{Format: FormatText})
			if err != nil {
				b.Fatalf("AppendText() returned error: %v", err)
			}

			benchBytesSink = out
		}
	})

	b.Run("EncodeBinary", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			out, err := AppendBinary(nil, benchBinaryDoc, EncodeOptions{Format: FormatBinary})
			if err != nil {
				b.Fatalf("AppendBinary() returned error: %v", err)
			}

			benchBytesSink = out
		}
	})
}

func BenchmarkTopLevelPreprocessFlow(b *testing.B) {
	b.ReportAllocs()

	b.Run("ParseAuto", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			doc, err := ParseAuto(benchBinaryIn)
			if err != nil {
				b.Fatalf("ParseAuto() returned error: %v", err)
			}

			benchDocSink = doc
		}
	})

	b.Run("WalkEvents", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			decoder := NewDecoder(bytes.NewReader(benchTextInput), DecodeOptions{Format: FormatText})
			for {
				event, err := decoder.WalkEvents()
				if err != nil {
					break
				}

				benchEventSink = event
			}
		}
	})

	b.Run("NextEvent", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			decoder := NewDecoder(bytes.NewReader(benchTextInput), DecodeOptions{Format: FormatText})
			for {
				event, err := decoder.NextEvent()
				if err != nil {
					break
				}

				benchEventSink = event
			}
		}
	})

	b.Run("NextEventBinary", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			decoder := NewDecoder(bytes.NewReader(benchBinaryIn), DecodeOptions{Format: FormatBinary})
			for {
				event, err := decoder.NextEvent()
				if err != nil {
					break
				}

				benchEventSink = event
			}
		}
	})
}

// benchMarshalStruct is a realistic struct for Marshal/Unmarshal benchmarks.
type benchMarshalStruct struct {
	Name    string `vdf:"name"`
	Version uint32 `vdf:"version"`
	Active  bool   `vdf:"active"`
	Tag     string `vdf:"tag"`
	Score   uint32 `vdf:"score"`
}

var benchMarshalInput = benchMarshalStruct{
	Name:    "server-bench",
	Version: 42,
	Active:  true,
	Tag:     "production",
	Score:   9999,
}

var benchMarshalDoc = func() *Document {
	doc, err := Marshal("Server", benchMarshalInput)
	if err != nil {
		panic(err)
	}
	return doc
}()

func BenchmarkReflectFlow(b *testing.B) {
	b.ReportAllocs()

	b.Run("Marshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			doc, err := Marshal("Server", benchMarshalInput)
			if err != nil {
				b.Fatalf("Marshal() error: %v", err)
			}
			benchDocSink = doc
		}
	})

	b.Run("Unmarshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var out benchMarshalStruct
			if err := Unmarshal(benchMarshalDoc, "Server", &out); err != nil {
				b.Fatalf("Unmarshal() error: %v", err)
			}
			benchAnySink = out
		}
	})

	b.Run("MarshalNested", func(b *testing.B) {
		type Inner struct {
			Host string `vdf:"host"`
			Port uint32 `vdf:"port"`
		}
		type Outer struct {
			Name string `vdf:"name"`
			DB   Inner  `vdf:"db"`
		}
		v := Outer{Name: "app", DB: Inner{Host: "localhost", Port: 5432}}
		for i := 0; i < b.N; i++ {
			doc, err := Marshal("Config", v)
			if err != nil {
				b.Fatalf("Marshal() error: %v", err)
			}
			benchDocSink = doc
		}
	})
}

func BenchmarkBuilderFlow(b *testing.B) {
	b.ReportAllocs()

	b.Run("FlatDocument", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			doc, err := NewBuilder("root").
				Set("name", "server").
				SetUint32("port", 2302).
				Set("map", "Namalsk").
				SetUint32("players", 60).
				Document()
			if err != nil {
				b.Fatalf("Document() error: %v", err)
			}
			benchDocSink = doc
		}
	})

	b.Run("NestedDocument", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			doc, err := NewBuilder("config").
				Set("name", "server").
				Object("network", func(b *Builder) {
					b.Set("host", "0.0.0.0").SetUint32("port", 2302)
				}).
				Object("game", func(b *Builder) {
					b.Set("map", "Namalsk").SetUint32("players", 60)
				}).
				Document()
			if err != nil {
				b.Fatalf("Document() error: %v", err)
			}
			benchDocSink = doc
		}
	})
}

func BenchmarkWriteEscaped(b *testing.B) {
	b.ReportAllocs()

	b.Run("NoEscape", func(b *testing.B) {
		w := &strings.Builder{}
		s := "simple value without special characters"
		for i := 0; i < b.N; i++ {
			w.Reset()
			_ = writeEscaped(w, s)
		}
		benchAnySink = w.String()
	})

	b.Run("WithEscape", func(b *testing.B) {
		w := &strings.Builder{}
		s := "value with\nnewlines\tand \"quotes\" and\\backslash"
		for i := 0; i < b.N; i++ {
			w.Reset()
			_ = writeEscaped(w, s)
		}
		benchAnySink = w.String()
	})
}
