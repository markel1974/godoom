package q3

import (
	"fmt"
	"io"
	"strings"

	"github.com/markel1974/godoom/mr_tech/generators/quake/interfaces"
)

// ShaderStage represents a stage within a shader, defining various rendering and texturing properties.
type ShaderStage struct {
	mapData    string
	clampMap   string
	animMap    []string
	videoMap   string
	blendSrc   string
	blendDst   string
	depthWrite bool
	depthFunc  string
	alphaFunc  string
	detail     bool
	tcGen      []string
	rgbGen     []string
	alphaGen   []string
	tcMods     [][]string
}

// IsAdditive determines if the shader stage uses additive blending by checking the blend source and destination factors.
func (st *ShaderStage) IsAdditive() bool {
	return st.blendSrc == "gl_one" && st.blendDst == "gl_one"
}

// Shader represents a graphics shader with specific properties, shader stages, and parameters for rendering operations.
type Shader struct {
	name            string
	surfaceParms    map[string]bool
	cull            string
	skyParms        []string
	fogParms        []string
	sort            string
	noPicMip        bool
	noMipmaps       bool
	polygonOffset   bool
	portal          bool
	entityMergeable bool
	tessSize        string
	deformVertexes  [][]string
	qerParms        map[string][]string
	q3mapParms      map[string][]string
	stages          []*ShaderStage
}

// NewShader creates a new Shader instance with the specified name and culling behavior.
func NewShader(name string, cull string) *Shader {
	return &Shader{
		name:         name,
		cull:         cull,
		surfaceParms: make(map[string]bool),
		qerParms:     make(map[string][]string),
		q3mapParms:   make(map[string][]string),
	}
}

// Shaders represents a collection of Shader objects organized in a map with shader names as keys.
type Shaders struct {
	container map[string]*Shader
}

// NewShaders initializes a new instance of the Shaders struct with an empty container map and returns its pointer.
func NewShaders() *Shaders {
	return &Shaders{
		container: make(map[string]*Shader),
	}
}

// Retrieve returns a map indicating shaders with at least one additive stage, keyed by shader name.
func (s *Shaders) Retrieve() map[string]bool {
	additive := make(map[string]bool)
	for name, sh := range s.container {
		for _, st := range sh.stages {
			if st.IsAdditive() {
				additive[name] = true
				break
			}
		}
	}
	return additive
}

// Reset clears the container map, effectively removing all stored Shader objects.
func (s *Shaders) Reset() {
	s.container = make(map[string]*Shader)
}

// Parse processes and loads shader files from the archive, filtering files by the "*.shader" pattern in the "scripts" folder.
func (s *Shaders) Parse(arc interfaces.IArchive) error {
	files, fErr := arc.ReadDirFilter("scripts", ".*\\.shader$")
	if fErr != nil {
		return fmt.Errorf("error reading scripts directory: %v", fErr)
	}

	for _, fName := range files {
		rs, err := arc.Open("scripts/" + fName)
		if err != nil {
			continue
		}
		data, err := io.ReadAll(rs)
		if err != nil {
			continue
		}
		s.parseData(string(data))
	}
	return nil
}

// shadersTokenType represents the type of tokens used in the shader parsing process.
type shadersTokenType int

// tokEOF represents the end-of-file token type.
// tokString represents a string token type.
// tokLBrace represents a left brace '{' token type.
// tokRBrace represents a right brace '}' token type.
// tokNewline represents a newline token type.
const (
	tokEOF shadersTokenType = iota
	tokString
	tokLBrace
	tokRBrace
	tokNewline
)

// ShadersToken represents a single lexical token in a shader source file.
// It contains the token's type and its associated text value.
type ShadersToken struct {
	kind shadersTokenType
	val  string
}

// ShadersScanner is a lexer for parsing shader definitions, processing characters into meaningful tokens.
type ShadersScanner struct {
	data []rune
	pos  int
}

// skipWhitespaceAndComments advances the scanner's position, skipping over whitespace and single-line or multi-line comments.
func (sc *ShadersScanner) skipWhitespaceAndComments() {
	for sc.pos < len(sc.data) {
		c := sc.data[sc.pos]
		if c == ' ' || c == '\t' || c == '\r' {
			sc.pos++
			continue
		}
		if c == '/' && sc.pos+1 < len(sc.data) && sc.data[sc.pos+1] == '/' {
			sc.pos += 2
			for sc.pos < len(sc.data) && sc.data[sc.pos] != '\n' {
				sc.pos++
			}
			continue
		}
		if c == '/' && sc.pos+1 < len(sc.data) && sc.data[sc.pos+1] == '*' {
			sc.pos += 2
			for sc.pos+1 < len(sc.data) {
				if sc.data[sc.pos] == '*' && sc.data[sc.pos+1] == '/' {
					sc.pos += 2
					break
				}
				sc.pos++
			}
			continue
		}
		break
	}
}

// nextToken scans the input data and returns the next ShadersToken detected at the current position.
func (sc *ShadersScanner) nextToken() ShadersToken {
	sc.skipWhitespaceAndComments()
	if sc.pos >= len(sc.data) {
		return ShadersToken{kind: tokEOF}
	}

	ct := sc.data[sc.pos]
	if ct == '\n' {
		sc.pos++
		return ShadersToken{kind: tokNewline, val: "\n"}
	}
	if ct == '{' {
		sc.pos++
		return ShadersToken{kind: tokLBrace, val: "{"}
	}
	if ct == '}' {
		sc.pos++
		return ShadersToken{kind: tokRBrace, val: "}"}
	}

	var sb strings.Builder
	if ct == '"' {
		sc.pos++
		for sc.pos < len(sc.data) && sc.data[sc.pos] != '"' {
			sb.WriteRune(sc.data[sc.pos])
			sc.pos++
		}
		if sc.pos < len(sc.data) {
			sc.pos++
		}
		return ShadersToken{kind: tokString, val: sb.String()}
	}

	for sc.pos < len(sc.data) {
		c := sc.data[sc.pos]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '{' || c == '}' || c == '"' {
			break
		}
		if c == '/' && sc.pos+1 < len(sc.data) && (sc.data[sc.pos+1] == '/' || sc.data[sc.pos+1] == '*') {
			break
		}
		sb.WriteRune(c)
		sc.pos++
	}
	return ShadersToken{kind: tokString, val: sb.String()}
}

// ShadersParser is a parser that processes shader file tokens using a ShadersScanner and maintains a lookahead token.
type ShadersParser struct {
	sc   *ShadersScanner
	peek *ShadersToken
}

// next retrieves the next token from the scanner or returns the peeked token if it exists, clearing the peek buffer.
func (p *ShadersParser) next() ShadersToken {
	if p.peek != nil {
		tok := *p.peek
		p.peek = nil
		return tok
	}
	return p.sc.nextToken()
}

// peekToken retrieves the next token without consuming it. If the token cache is empty, it fetches the next token.
func (p *ShadersParser) peekToken() ShadersToken {
	if p.peek == nil {
		tok := p.sc.nextToken()
		p.peek = &tok
	}
	return *p.peek
}

// skipUntil advances the parser until a token of the specified type or EOF is encountered.
func (p *ShadersParser) skipUntil(typ shadersTokenType) {
	for {
		tok := p.next()
		if tok.kind == typ || tok.kind == tokEOF {
			break
		}
	}
}

// consumeLineArgs collects and returns all arguments from the current line until a newline, brace, or EOF is encountered.
func (p *ShadersParser) consumeLineArgs() []string {
	var args []string
	for {
		t := p.peekToken()
		if t.kind == tokNewline || t.kind == tokEOF || t.kind == tokLBrace || t.kind == tokRBrace {
			if t.kind == tokNewline {
				p.next()
			}
			break
		}
		tok := p.next()
		args = append(args, tok.val)
	}
	return args
}

// parseData parses shader definitions from the provided data string and adds them to the Shaders container.
func (s *Shaders) parseData(data string) {
	p := &ShadersParser{sc: &ShadersScanner{data: []rune(data)}}

	for {
		// 1. Skip newlines at root level
		tok := p.next()
		for tok.kind == tokNewline {
			tok = p.next()
		}

		if tok.kind == tokEOF {
			break
		}

		if tok.kind != tokString {
			continue // Unmatched braces at root level, skip
		}

		shaderName := strings.ToLower(tok.val)

		// Expect LBrace (skipping newlines)
		brTok := p.next()
		for brTok.kind == tokNewline {
			brTok = p.next()
		}

		if brTok.kind != tokLBrace {
			if brTok.kind == tokString {
				p.peek = &brTok
			}
			continue
		}

		shader := NewShader(shaderName, "front")

		// Parse shader body
		for {
			t := p.next()
			for t.kind == tokNewline {
				t = p.next()
			}

			if t.kind == tokEOF || t.kind == tokRBrace {
				break
			}

			if t.kind == tokLBrace {
				// Parse stage block
				stage := &ShaderStage{depthWrite: true}
				for {
					st := p.next()
					for st.kind == tokNewline {
						st = p.next()
					}

					if st.kind == tokEOF || st.kind == tokRBrace {
						break
					}
					if st.kind == tokLBrace {
						p.skipUntil(tokRBrace)
						continue
					}

					if st.kind == tokString {
						cmd := strings.ToLower(st.val)
						args := p.consumeLineArgs()

						if cmd == "map" && len(args) > 0 {
							stage.mapData = args[0]
						} else if cmd == "clampmap" && len(args) > 0 {
							stage.clampMap = args[0]
						} else if cmd == "animmap" && len(args) > 0 {
							stage.animMap = args
						} else if cmd == "videomap" && len(args) > 0 {
							stage.videoMap = args[0]
						} else if cmd == "blendfunc" && len(args) > 0 {
							arg1 := strings.ToLower(args[0])
							if arg1 == "add" {
								stage.blendSrc, stage.blendDst, stage.depthWrite = "gl_one", "gl_one", false
							} else if arg1 == "filter" {
								stage.blendSrc, stage.blendDst, stage.depthWrite = "gl_dst_color", "gl_zero", false
							} else if arg1 == "blend" {
								stage.blendSrc, stage.blendDst, stage.depthWrite = "gl_src_alpha", "gl_one_minus_src_alpha", false
							} else if len(args) > 1 {
								stage.blendSrc, stage.blendDst, stage.depthWrite = arg1, strings.ToLower(args[1]), false
							}
						} else if cmd == "alphafunc" && len(args) > 0 {
							stage.alphaFunc = strings.ToLower(args[0])
						} else if cmd == "depthfunc" && len(args) > 0 {
							stage.depthFunc = strings.ToLower(args[0])
						} else if cmd == "depthwrite" {
							stage.depthWrite = true
						} else if cmd == "detail" {
							stage.detail = true
						} else if cmd == "tcmod" {
							stage.tcMods = append(stage.tcMods, args)
						} else if cmd == "tcgen" {
							stage.tcGen = args
						} else if cmd == "rgbgen" {
							stage.rgbGen = args
						} else if cmd == "alphagen" {
							stage.alphaGen = args
						}
					}
				}
				shader.stages = append(shader.stages, stage)
				continue
			}

			if t.kind == tokString {
				cmd := strings.ToLower(t.val)
				args := p.consumeLineArgs()

				if cmd == "surfaceparm" && len(args) > 0 {
					shader.surfaceParms[strings.ToLower(args[0])] = true
				} else if cmd == "cull" && len(args) > 0 {
					shader.cull = strings.ToLower(args[0])
				} else if cmd == "skyparms" {
					shader.skyParms = args
				} else if cmd == "fogparms" {
					shader.fogParms = args
				} else if cmd == "sort" && len(args) > 0 {
					shader.sort = args[0]
				} else if cmd == "nopicmip" {
					shader.noPicMip = true
				} else if cmd == "nomipmaps" {
					shader.noMipmaps = true
				} else if cmd == "polygonoffset" {
					shader.polygonOffset = true
				} else if cmd == "portal" {
					shader.portal = true
				} else if cmd == "entitymergable" {
					shader.entityMergeable = true
				} else if cmd == "tesssize" && len(args) > 0 {
					shader.tessSize = args[0]
				} else if cmd == "deformvertexes" {
					shader.deformVertexes = append(shader.deformVertexes, args)
				} else if strings.HasPrefix(cmd, "qer_") {
					shader.qerParms[cmd] = args
				} else if strings.HasPrefix(cmd, "q3map_") {
					shader.q3mapParms[cmd] = args
				}
			}
		}

		s.container[shader.name] = shader
	}
}
