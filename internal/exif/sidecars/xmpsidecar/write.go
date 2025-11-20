package xmpsidecar

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
)

const (
	namespaceRDF     = "http://www.w3.org/1999/02/22-rdf-syntax-ns#"
	namespaceDigiKam = "http://www.digikam.org/ns/1.0/"
	namespaceX       = "adobe:ns:meta/"
)

type namespaceDecl struct {
	uri      string
	previous string
	hadPrev  bool
}

type namespaceState struct {
	stack       [][]namespaceDecl
	uriToPrefix map[string]string
}

func newNamespaceState() *namespaceState {
	return &namespaceState{
		stack: nil,
		uriToPrefix: map[string]string{
			"http://www.w3.org/XML/1998/namespace": "xml",
		},
	}
}

func (ns *namespaceState) push(start xml.StartElement) {
	var decls []namespaceDecl
	for _, attr := range start.Attr {
		if attr.Name.Space == "xmlns" || (attr.Name.Space == "" && attr.Name.Local == "xmlns") {
			prefix := attr.Name.Local
			if attr.Name.Space == "" && attr.Name.Local == "xmlns" {
				prefix = ""
			}
			uri := attr.Value
			prev, ok := ns.uriToPrefix[uri]
			decls = append(decls, namespaceDecl{uri: uri, previous: prev, hadPrev: ok})
			ns.uriToPrefix[uri] = prefix
		}
	}
	ns.stack = append(ns.stack, decls)
}

func (ns *namespaceState) pop() {
	if len(ns.stack) == 0 {
		return
	}
	decls := ns.stack[len(ns.stack)-1]
	ns.stack = ns.stack[:len(ns.stack)-1]
	for i := len(decls) - 1; i >= 0; i-- {
		decl := decls[i]
		if decl.hadPrev {
			ns.uriToPrefix[decl.uri] = decl.previous
		} else {
			delete(ns.uriToPrefix, decl.uri)
		}
	}
}

func (ns *namespaceState) encodeStart(encoder *xml.Encoder, start xml.StartElement) error {
	ns.push(start)
	return encoder.EncodeToken(ns.convertStart(start))
}

func (ns *namespaceState) encodeEnd(encoder *xml.Encoder, end xml.EndElement) error {
	converted := ns.convertEnd(end)
	ns.pop()
	return encoder.EncodeToken(converted)
}

func (ns *namespaceState) convertStart(start xml.StartElement) xml.StartElement {
	converted := xml.StartElement{
		Name: ns.convertName(start.Name),
		Attr: make([]xml.Attr, len(start.Attr)),
	}
	for i, attr := range start.Attr {
		converted.Attr[i] = ns.convertAttr(attr)
	}
	return converted
}

func (ns *namespaceState) convertEnd(end xml.EndElement) xml.EndElement {
	return xml.EndElement{Name: ns.convertName(end.Name)}
}

func (ns *namespaceState) convertAttr(attr xml.Attr) xml.Attr {
	switch {
	case attr.Name.Space == "" && attr.Name.Local == "xmlns":
		return xml.Attr{Name: xml.Name{Local: "xmlns"}, Value: attr.Value}
	case attr.Name.Space == "xmlns":
		local := attr.Name.Local
		if local == "" {
			return xml.Attr{Name: xml.Name{Local: "xmlns"}, Value: attr.Value}
		}
		return xml.Attr{Name: xml.Name{Local: "xmlns:" + local}, Value: attr.Value}
	default:
		return xml.Attr{Name: ns.convertName(attr.Name), Value: attr.Value}
	}
}

func (ns *namespaceState) convertName(name xml.Name) xml.Name {
	if name.Local == "" {
		return xml.Name{}
	}
	if name.Space == "" {
		return xml.Name{Local: name.Local}
	}
	if prefix, ok := ns.uriToPrefix[name.Space]; ok {
		if prefix == "" {
			return xml.Name{Local: name.Local}
		}
		return xml.Name{Local: prefix + ":" + name.Local}
	}
	return xml.Name{Local: name.Local}
}

// UpdateTags returns a copy of the provided XMP document with the TagsList updated to contain the
// provided tags. When no source is supplied, a minimal XMP document is generated.
func UpdateTags(source []byte, tags []string) ([]byte, error) {
	if len(source) == 0 {
		if len(tags) == 0 {
			return nil, nil
		}
		return buildNewSidecar(tags)
	}

	updated, err := rewriteTags(source, tags)
	if err != nil {
		if len(tags) == 0 {
			return source, nil
		}
		return buildNewSidecar(tags)
	}
	return updated, nil
}

func rewriteTags(source []byte, tags []string) ([]byte, error) {
	decoder := xml.NewDecoder(bytes.NewReader(source))
	buffer := &bytes.Buffer{}
	encoder := xml.NewEncoder(buffer)
	encoder.Indent("", "  ")
	ns := newNamespaceState()

	var (
		insideTagsList bool
		nestedDepth    int
		tagsListFound  bool
	)

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		switch tok := token.(type) {
		case xml.StartElement:
			if tok.Name.Space == namespaceDigiKam && tok.Name.Local == "TagsList" {
				insideTagsList = true
				nestedDepth = 0
				tagsListFound = true
				if err = ns.encodeStart(encoder, tok); err != nil {
					return nil, err
				}
				continue
			}
			if insideTagsList {
				ns.push(tok)
				nestedDepth++
				continue
			}
			if err = ns.encodeStart(encoder, tok); err != nil {
				return nil, err
			}
		case xml.EndElement:
			if insideTagsList {
				if nestedDepth > 0 {
					nestedDepth--
					ns.pop()
					continue
				}
				if err = writeSeq(encoder, ns, tags); err != nil {
					return nil, err
				}
				if err = ns.encodeEnd(encoder, tok); err != nil {
					return nil, err
				}
				insideTagsList = false
				continue
			}
			if tok.Name.Space == namespaceRDF && tok.Name.Local == "RDF" && !tagsListFound && len(tags) > 0 {
				if err = writeDescription(encoder, ns, tags); err != nil {
					return nil, err
				}
				tagsListFound = true
			}
			if err = ns.encodeEnd(encoder, tok); err != nil {
				return nil, err
			}
		case xml.CharData:
			if insideTagsList {
				continue
			}
			if err = encoder.EncodeToken(tok); err != nil {
				return nil, err
			}
		case xml.Comment:
			if insideTagsList {
				continue
			}
			if err = encoder.EncodeToken(tok); err != nil {
				return nil, err
			}
		case xml.Directive, xml.ProcInst:
			if err = encoder.EncodeToken(tok); err != nil {
				return nil, err
			}
		}
	}

	if insideTagsList {
		return nil, errors.New("unterminated TagsList element")
	}
	if !tagsListFound && len(tags) > 0 {
		return nil, errors.New("missing RDF Description to attach TagsList")
	}

	if err := encoder.Flush(); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func buildNewSidecar(tags []string) ([]byte, error) {
	buffer := &bytes.Buffer{}
	_, _ = buffer.WriteString("<?xpacket begin='\ufeff' id='W5M0MpCehiHzreSzNTczkc9d'?>\n")

	encoder := xml.NewEncoder(buffer)
	encoder.Indent("", "  ")
	ns := newNamespaceState()

	xmpStart := xml.StartElement{
		Name: xml.Name{Space: namespaceX, Local: "xmpmeta"},
		Attr: []xml.Attr{
			{Name: xml.Name{Space: "xmlns", Local: "x"}, Value: namespaceX},
		},
	}
	if err := ns.encodeStart(encoder, xmpStart); err != nil {
		return nil, err
	}

	rdfStart := xml.StartElement{
		Name: xml.Name{Space: namespaceRDF, Local: "RDF"},
		Attr: []xml.Attr{
			{Name: xml.Name{Space: "xmlns", Local: "rdf"}, Value: namespaceRDF},
		},
	}
	if err := ns.encodeStart(encoder, rdfStart); err != nil {
		return nil, err
	}

	if err := writeDescription(encoder, ns, tags); err != nil {
		return nil, err
	}

	if err := ns.encodeEnd(encoder, rdfStart.End()); err != nil {
		return nil, err
	}
	if err := ns.encodeEnd(encoder, xmpStart.End()); err != nil {
		return nil, err
	}

	if err := encoder.Flush(); err != nil {
		return nil, err
	}

	_, _ = buffer.WriteString("\n<?xpacket end='w'?>")
	return buffer.Bytes(), nil
}

func writeDescription(encoder *xml.Encoder, ns *namespaceState, tags []string) error {
	descStart := xml.StartElement{
		Name: xml.Name{Space: namespaceRDF, Local: "Description"},
		Attr: []xml.Attr{
			{Name: xml.Name{Space: namespaceRDF, Local: "about"}, Value: ""},
			{Name: xml.Name{Space: "xmlns", Local: "digiKam"}, Value: namespaceDigiKam},
		},
	}
	if err := ns.encodeStart(encoder, descStart); err != nil {
		return err
	}

	tagsStart := xml.StartElement{Name: xml.Name{Space: namespaceDigiKam, Local: "TagsList"}}
	if err := ns.encodeStart(encoder, tagsStart); err != nil {
		return err
	}

	if err := writeSeq(encoder, ns, tags); err != nil {
		return err
	}

	if err := ns.encodeEnd(encoder, tagsStart.End()); err != nil {
		return err
	}

	return ns.encodeEnd(encoder, descStart.End())
}

func writeSeq(encoder *xml.Encoder, ns *namespaceState, tags []string) error {
	seqStart := xml.StartElement{Name: xml.Name{Space: namespaceRDF, Local: "Seq"}}
	if err := ns.encodeStart(encoder, seqStart); err != nil {
		return err
	}

	for _, tag := range tags {
		liStart := xml.StartElement{Name: xml.Name{Space: namespaceRDF, Local: "li"}}
		if err := ns.encodeStart(encoder, liStart); err != nil {
			return err
		}
		if err := encoder.EncodeToken(xml.CharData(tag)); err != nil {
			return err
		}
		if err := ns.encodeEnd(encoder, liStart.End()); err != nil {
			return err
		}
	}

	return ns.encodeEnd(encoder, seqStart.End())
}
