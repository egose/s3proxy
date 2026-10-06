package namespace

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"testing/iotest"
)

func fragmentedText(text, mode string, width int) string {
	var out bytes.Buffer
	runes := []rune(text)
	for i := 0; i < len(runes); i += width {
		part := string(runes[i:min(i+width, len(runes))])
		if mode == "CDATA" || (mode == "Mixed" && i/width%2 == 0) {
			out.WriteString("<![CDATA[" + part + "]]>")
		} else {
			_ = xml.EscapeText(&out, []byte(part))
		}
		if mode == "Comment" || mode == "Mixed" {
			out.WriteString("<!-- ignored &amp; <hidden>other/secret</hidden> -->")
		}
	}
	return out.String()
}

func fragmentedListing(mode string) string {
	leaf := func(name, text string) string {
		return "<" + name + ">" + fragmentedText(text, mode, 1) + "</" + name + ">"
	}
	return `<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">` +
		leaf("Name", "store") + leaf("Prefix", "tenant/") + leaf("Delimiter", "/") +
		leaf("IsTruncated", "true") + leaf("ContinuationToken", "opaque&<雪+%2F") +
		leaf("NextContinuationToken", strings.Repeat("a&<雪+%2F", 256)) +
		"<Contents>" + leaf("Key", "tenant/a&<雪+%2F") + leaf("ETag", "") +
		"<Owner>" + leaf("ID", "owner&雪") + leaf("DisplayName", " display\tname\n ") + "</Owner>" +
		leaf("ChecksumAlgorithm", "SHA256") + leaf("ChecksumAlgorithm", "CRC32") + "</Contents>" +
		"<Contents>" + leaf("Key", "tenant/second") + leaf("ETag", "second-etag") + "</Contents>" +
		"<CommonPrefixes>" + leaf("Prefix", "tenant/dir&雪/") + "</CommonPrefixes></ListBucketResult>"
}

func TestTransformFragmentedTextEquivalent(t *testing.T) {
	m := Mapping{VisibleBucket: "visible", Prefix: "tenant/"}
	query := url.Values{"delimiter": {"/"}, "continuation-token": {"opaque&<雪+%2F"}}
	body := &trackedBody{Reader: strings.NewReader(fragmentedListing("Plain"))}
	want, err := m.Transform(context.Background(), body, "store", query)
	if err != nil || body.closed.Load() != 1 {
		t.Fatalf("plain: err=%v closes=%d", err, body.closed.Load())
	}
	for _, field := range []string{"<Name>visible</Name>", "<Prefix></Prefix>", "<Key>a&amp;&lt;雪+%2F</Key>", "<ETag></ETag>", "<Key>second</Key>", "<Prefix>dir&amp;雪/</Prefix>"} {
		if !bytes.Contains(want, []byte(field)) {
			t.Fatalf("missing virtual field %q", field)
		}
	}
	for _, mode := range []string{"Plain", "Comment", "CDATA", "Mixed"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			input := fragmentedListing(mode)
			input = strings.ReplaceAll(input, "&amp;", "&#38;")
			body := &trackedBody{Reader: iotest.OneByteReader(strings.NewReader(input))}
			got, err := m.Transform(context.Background(), body, "store", query)
			if err != nil || !bytes.Equal(got, want) || body.closed.Load() != 1 {
				t.Fatalf("equivalent=%v err=%v closes=%d", bytes.Equal(got, want), err, body.closed.Load())
			}
		})
	}
}

func TestTransformFragmentedTextRejectsAndCloses(t *testing.T) {
	for _, mode := range []string{"Plain", "Comment", "CDATA", "Mixed"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			text := fragmentedText("tenant/a&雪", mode, 1)
			input := strings.Replace(validXML, "tenant/a", text, 1)
			for name, invalid := range map[string]string{
				"unclosed root":       strings.TrimSuffix(input, "</ListBucketResult>"),
				"mismatched end":      strings.Replace(input, "</Key>", "</ETag>", 1),
				"unclosed comment":    strings.Replace(input, "</Key>", "<!--broken</Key>", 1),
				"unclosed CDATA":      strings.Replace(input, "</Key>", "<![CDATA[broken</Key>", 1),
				"bad entity":          strings.Replace(input, "</Key>", "&unknown;</Key>", 1),
				"foreign namespace":   strings.Replace(input, "<Key>", `<Key xmlns="other">`, 1),
				"empty namespace":     strings.Replace(input, "<Key>", `<Key xmlns="">`, 1),
				"nested foreign leaf": strings.Replace(input, "</Key>", `<x:Key xmlns:x="other">a</x:Key></Key>`, 1),
				"duplicate field":     strings.Replace(input, "</Key>", "</Key><Key>"+text+"</Key>", 1),
				"container text":      strings.Replace(input, "<Contents>", "<Contents>"+text, 1),
				"trailing text":       input + text,
				"sibling key":         strings.Replace(input, text, fragmentedText("tenant-other/secret", mode, 1), 1),
				"sibling prefix":      strings.Replace(input, "<Contents>", "<CommonPrefixes><Prefix>"+fragmentedText("other/", mode, 1)+"</Prefix></CommonPrefixes><Contents>", 1),
			} {
				t.Run(name, func(t *testing.T) {
					body := &trackedBody{Reader: iotest.OneByteReader(strings.NewReader(invalid))}
					data, err := (Mapping{VisibleBucket: "visible", Prefix: "tenant/"}).Transform(context.Background(), body, "store", nil)
					if !errors.Is(err, ErrResponse) || data != nil || body.closed.Load() != 1 {
						t.Fatalf("data=%q err=%v closes=%d", data, err, body.closed.Load())
					}
				})
			}
		})
	}
}

func BenchmarkParseResultFragmentedText(b *testing.B) {
	for _, mode := range []string{"Plain", "Comment", "CDATA", "Mixed"} {
		for _, size := range []int{1 << 10, 8 << 10, 64 << 10} {
			b.Run(fmt.Sprintf("%s/%dKiB", mode, size>>10), func(b *testing.B) {
				text := strings.Repeat("abcdefgh", size/8)
				input := []byte(strings.Replace(validXML, "etag", fragmentedText(text, mode, 8), 1))
				ctx := context.Background()
				b.ReportAllocs()
				b.SetBytes(int64(len(input)))
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					root, err := parseResult(ctx, input)
					if err != nil || root.child("Contents").value("ETag") != text {
						b.Fatalf("incorrect parsed text: %v", err)
					}
				}
			})
		}
	}
}
