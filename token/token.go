package token

type TokenType string

const (
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"

	// Identifiers + literals
	IDENT  = "IDENT" // add, foobar, x, y, ...
	INT    = "INT"   // 1343456
	FLOAT  = "FLOAT"
	STRING = "STRING" // "foobar"

	INC = "++"
	DEC = "--"

	// Operators
	ASSIGN   = "="
	PLUS     = "+"
	MINUS    = "-"
	BANG     = "!"
	ASTERISK = "*"
	SLASH    = "/"

	LT = "<"
	GT = ">"
	LE = "<="
	GE = ">="

	EQ     = "=="
	NOT_EQ = "!="

	// Delimiters
	COMMA     = ","
	SEMICOLON = ";"
	COLON     = ":"

	LPAREN   = "("
	RPAREN   = ")"
	LBRACE   = "{"
	RBRACE   = "}"
	LBRACKET = "["
	RBRACKET = "]"

	// Keywords
	FUNCTION = "FUNCTION"
	LET      = "LET"
	MAY      = "MAY"
	TRUE     = "TRUE"
	FALSE    = "FALSE"
	NULL     = "NULL"
	IF       = "IF"
	ELSE     = "ELSE"
	WHILE    = "WHILE"
	RETURN   = "RETURN"

	MINT = "MINT"
	LINT = "LINT"
	MFLT = "MFLT"
	LFLT = "LFLT"
	MSTR = "MSTR"
	LSTR = "LSTR"
	MARR = "MARR"
	LARR = "LARR"
	MDCT = "MDCT"
	LDCT = "LDCT"
	ANY  = "ANY"
)

type Token struct {
	Type    TokenType
	Literal string
}

var keywords = map[string]TokenType{
	"fn":     FUNCTION,
	"let":    LET,
	"may":    MAY,
	"true":   TRUE,
	"false":  FALSE,
	"null":   NULL,
	"if":     IF,
	"else":   ELSE,
	"while":  WHILE,
	"return": RETURN,

	"mint":        MINT,
	"lint":        LINT,
	"endeavouros": MINT,
	"mflt":        MFLT,
	"lflt":        LFLT,
	"mstr":        MSTR,
	"lstr":        LSTR,
	"marr":        MARR,
	"larr":        LARR,
	"larry":       LARR,
	"mdct":        MDCT,
	"ldct":        LDCT,
	"any":         ANY,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
