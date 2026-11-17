package searchindex

// stopwords are dropped by the tokenizer. English only for now.
var stopwords = map[string]bool{
	"a": true, "an": true, "and": true, "are": true, "as": true, "at": true,
	"be": true, "but": true, "by": true, "for": true, "from": true, "has": true,
	"have": true, "he": true, "in": true, "is": true, "it": true, "its": true,
	"of": true, "on": true, "or": true, "that": true, "the": true, "this": true,
	"to": true, "was": true, "were": true, "will": true, "with": true, "you": true,
	"your": true, "our": true, "we": true, "they": true, "them": true, "their": true,
	"than": true, "then": true, "there": true, "these": true, "those": true, "not": true,
	"no": true, "yes": true,
}
