package vdf

import (
	"bytes"
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

	b.Run("EventStream", func(b *testing.B) {
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
}
