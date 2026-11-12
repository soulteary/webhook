package hook

import "testing"

// 期望值为空（如 getenv 读到未设置的变量）时，空参数不能通过 value 规则。
func TestMatchRule_EmptyValueNeverMatches(t *testing.T) {
	rule := MatchRule{Type: MatchValue, Value: "", Parameter: Argument{Source: "url", Name: "token"}}
	for _, query := range []map[string]interface{}{{"token": ""}, {"token": "anything"}} {
		ok, err := rule.Evaluate(&Request{Query: query})
		if err != nil {
			t.Fatalf("Evaluate(%v): unexpected error: %v", query, err)
		}
		if ok {
			t.Fatalf("Evaluate(%v) matched an empty configured value", query)
		}
	}

	rule.Value = "s3cret"
	ok, err := rule.Evaluate(&Request{Query: map[string]interface{}{"token": "s3cret"}})
	if err != nil || !ok {
		t.Fatalf("non-empty value should still match: ok=%v err=%v", ok, err)
	}
}
