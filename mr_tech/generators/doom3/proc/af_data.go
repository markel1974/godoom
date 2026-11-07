package proc

import "strings"

// AFBody represents a single rigid body segment from a Doom 3 .af (Articulated Figure) ragdoll file.
type AFBody struct {
	Name            string
	Joint           string
	ContainedJoints string
}

// AFDecl represents a parsed articulatedFigure declaration from an .af file.
type AFDecl struct {
	Name   string
	Bodies []AFBody
}

type AFData struct {
	AFDecls map[string]*AFDecl
}

func NewAFData() *AFData {
	return &AFData{
		AFDecls: make(map[string]*AFDecl),
	}
}

// Parse parses articulatedFigure and body blocks from raw .af file bytes.
func (d *AFData) Parse(fileStem string, data []byte) {
	lexer := NewLexer(data)
	declName := fileStem
	var bodies []AFBody

	for {
		tok, err := lexer.NextToken()
		if err != nil {
			break
		}
		tokLower := strings.ToLower(StripQuotes(tok))
		if tokLower == "articulatedfigure" {
			if nameTok, err := lexer.NextToken(); err == nil && nameTok != "{" {
				declName = strings.ToLower(StripQuotes(nameTok))
			}
			continue
		}
		if tokLower == "body" {
			nameTok, err := lexer.NextToken()
			if err != nil {
				break
			}
			if peek, err := lexer.PeekToken(); err != nil || peek != "{" {
				continue
			}
			_ = lexer.Expect("{")
			body := AFBody{
				Name: strings.ToLower(StripQuotes(nameTok)),
			}
			depth := 1
			parenDepth := 0
			for depth > 0 {
				bTok, err := lexer.NextToken()
				if err != nil {
					break
				}
				if bTok == "{" {
					depth++
					continue
				}
				if bTok == "}" {
					depth--
					continue
				}
				if bTok == "(" {
					parenDepth++
					continue
				}
				if bTok == ")" {
					if parenDepth > 0 {
						parenDepth--
					}
					continue
				}
				if depth != 1 || parenDepth > 0 {
					continue
				}
				bLower := strings.ToLower(StripQuotes(bTok))
				switch bLower {
				case "joint":
					if peek, err := lexer.PeekToken(); err == nil && peek != "(" && peek != "{" && peek != "}" {
						if jTok, err := lexer.NextToken(); err == nil {
							body.Joint = StripQuotes(jTok)
						}
					}
				case "containedjoints":
					if cjTok, err := lexer.NextToken(); err == nil && cjTok != "{" && cjTok != "}" {
						body.ContainedJoints = StripQuotes(cjTok)
					}
				}
			}
			bodies = append(bodies, body)
		}
	}

	if len(bodies) > 0 {
		afDecl := &AFDecl{
			Name:   declName,
			Bodies: bodies,
		}
		d.AFDecls[declName] = afDecl
		if len(fileStem) > 0 {
			d.AFDecls[fileStem] = afDecl
		}
	}
}

// ResolveBodies looks up the parsed AFBody slice for an articulated figure name or .af path.
func (d *AFData) ResolveBodies(ragdoll string) ([]AFBody, bool) {
	if len(ragdoll) == 0 {
		return nil, false
	}
	clean := strings.ToLower(strings.TrimSpace(StripQuotes(ragdoll)))
	clean = strings.TrimPrefix(clean, "af/")
	clean = strings.TrimSuffix(clean, ".af")
	if decl, ok := d.AFDecls[clean]; ok && len(decl.Bodies) > 0 {
		return decl.Bodies, true
	}
	return nil, false
}
