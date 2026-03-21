package tagger

import "strings"

// patternRule maps a tag name to the keyword signals that trigger it.
// Order matters: tags are emitted in this declaration order.
var patternRules = []struct {
	tag      string
	keywords []string
}{
	{"retry", []string{
		"retry", "Retry", "backoff", "BackOff", "MaxRetries", "maxRetries", "retryCount",
	}},
	{"http_call", []string{
		"http.Get", "http.Post", "http.Do", "http.NewRequest",
		"fetch(", "requests.get", "requests.post", "HttpClient", "urllib", "axios",
	}},
	{"db_access", []string{
		"db.Query", "db.Exec", "sql.Open", "sql.DB",
		"SELECT ", "INSERT ", "UPDATE ", "DELETE ",
		"cursor.execute", ".FindAll", ".findById",
	}},
	{"validation", []string{
		"validate", "Validate", "IsValid", "isValid", "ErrInvalid",
		"assert(", "Must(", "required",
	}},
	{"mapping", []string{
		"transform", "Transform", "convert", "Convert",
		"ToDTO", "FromDTO", "toMap", "json.Marshal", "json.Unmarshal",
	}},
	{"transaction", []string{
		".Begin(", ".Commit(", ".Rollback(", "Transaction(",
		"tx.", "BEGIN TRANSACTION", "COMMIT", "ROLLBACK",
	}},
	{"caching", []string{
		"cache.", "Cache{", "redis.", "Redis(", "memcache",
		".TTL", "sync.Map", "expire", "Expire",
	}},
	{"concurrency", []string{
		"go func", "WaitGroup", "sync.Mutex", "chan ", "<-chan",
		"select {", "atomic.", "async ", "await ", "Promise.", "Thread(", ".Lock()",
	}},
	{"error_wrapping", []string{
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
				tags = append(tags, rule.tag)
				break // one keyword match is enough to apply the tag
			}
		}
	}
	return tags
}
