package tagger

import (
	"strings"

	"github.com/lukse/doppel/internal/ontology"
)

// patternRules maps a concept to the keyword signals that trigger it.
// Order matters: tags are emitted in this declaration order.
//
// The rules name ontology terms rather than bare strings so the two cannot
// drift apart: a rule for a concept that does not exist stops compiling, and
// tagger_test enforces the other direction, that every concrete concept has
// exactly one rule. Tag still emits the term IDs as plain strings, and those
// IDs are the tag names this tool has always printed, so output is unchanged.
var patternRules = []struct {
	concept  ontology.TermID
	keywords []string
}{
	{ontology.ConRetry, []string{
		"retry", "Retry", "backoff", "BackOff", "MaxRetries", "maxRetries", "retryCount",
	}},
	{ontology.ConHTTPCall, []string{
		"http.Get", "http.Post", "http.Do", "http.NewRequest",
		"fetch(", "requests.get", "requests.post", "HttpClient", "urllib", "axios",
	}},
	{ontology.ConDBAccess, []string{
		"db.Query", "db.Exec", "sql.Open", "sql.DB",
		"SELECT ", "INSERT ", "UPDATE ", "DELETE ",
		"cursor.execute", ".FindAll", ".findById",
	}},
	{ontology.ConValidation, []string{
		"validate", "Validate", "IsValid", "isValid", "ErrInvalid",
		"assert(", "Must(", "required",
	}},
	{ontology.ConMapping, []string{
		"transform", "Transform", "convert", "Convert",
		"ToDTO", "FromDTO", "toMap", "json.Marshal", "json.Unmarshal",
	}},
	{ontology.ConTransaction, []string{
		".Begin(", ".Commit(", ".Rollback(", "Transaction(",
		"tx.", "BEGIN TRANSACTION", "COMMIT", "ROLLBACK",
	}},
	{ontology.ConCaching, []string{
		"cache.", "Cache{", "redis.", "Redis(", "memcache",
		".TTL", "sync.Map", "expire", "Expire",
	}},
	{ontology.ConConcurrency, []string{
		"go func", "WaitGroup", "sync.Mutex", "chan ", "<-chan",
		"select {", "atomic.", "async ", "await ", "Promise.", "Thread(", ".Lock()",
	}},
	{ontology.ConErrorWrapping, []string{
		"fmt.Errorf", "errors.Wrap", "errors.As", "errors.Is",
		`%w"`, "WithMessage(", "WithStack(", "Wrapf(",
	}},
}

// Tag returns the pattern labels detected in the function body.
// Tags are returned in a deterministic order matching the rule declaration order.
func Tag(body string) []string {
	var tags []string
	for _, rule := range patternRules {
		for _, kw := range rule.keywords {
			if strings.Contains(body, kw) {
				tags = append(tags, string(rule.concept))
				break // one keyword match is enough to apply the tag
			}
		}
	}
	return tags
}
