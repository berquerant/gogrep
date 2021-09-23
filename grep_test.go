package gogrep_test

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/berquerant/gogrep"
	"github.com/stretchr/testify/assert"
)

func dupStrings(n int, seeds ...string) []string {
	r := make([]string, len(seeds)*n)
	for i := 0; i < len(r); i++ {
		r[i] = seeds[i%len(seeds)]
	}
	return r
}

func TestGrepper(t *testing.T) {
	t.Run("already canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.TODO())
		cancel()
		_, err := gogrep.New().Grep(ctx, "ra", nil)
		assert.NotNil(t, err)
	})

	t.Run("invalid regex", func(t *testing.T) {
		_, err := gogrep.New().Grep(context.TODO(), "?", nil)
		assert.NotNil(t, err)
	})

	for _, tc := range []*struct {
		title string
		regex string
		input []string
		want  []string
	}{
		{
			title: "no input",
			regex: "vanity",
		},
		{
			title: "not matched",
			regex: "vanity",
			input: []string{"empty"},
		},
		{
			title: "matched",
			regex: "vanity",
			input: []string{"vanity"},
			want:  []string{"vanity"},
		},
		{
			title: "long input not matched",
			regex: "vanity",
			input: dupStrings(300, "empty"),
		},
		{
			title: "long input matched",
			regex: "vanity",
			input: dupStrings(300, "vanity"),
			want:  dupStrings(300, "vanity"),
		},
		{
			title: "long input matched partially",
			regex: "afford|deny",
			input: dupStrings(300, "empty", "afford", "vanity", "deny"),
			want:  dupStrings(300, "afford", "deny"),
		},
		{
			title: "long input matched partially lines",
			regex: "afford|prove|those$",
			input: dupStrings(300, "one of those days", "affordance", "vanitas", "prove all things"),
			want:  dupStrings(300, "affordance", "prove all things"),
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			source := strings.NewReader(strings.Join(tc.input, "\n"))
			resultC, err := gogrep.New().Grep(context.TODO(), tc.regex, source)
			if err != nil {
				t.Fatal(err)
			}
			got := []string{}
			for matched := range resultC {
				assert.Nil(t, matched.Err())
				got = append(got, matched.Text())
			}
			assert.Equal(t, len(tc.want), len(got))
			sort.Strings(tc.want)
			sort.Strings(got)
			for i, w := range tc.want {
				g := got[i]
				assert.Equal(t, w, g)
			}
		})
	}
}

func BenchmarkGrepper(b *testing.B) {
	for i := 0; i <= 5; i++ {
		threads := 1 << i
		b.Run(fmt.Sprintf("with %d threads", threads), func(b *testing.B) {
			data := strings.NewReader(strings.Join(dupStrings(b.N, "allocation", "freeable", "cached", "dirty", "flush memory", "NAND", "ready to write"), "\n"))
			b.ResetTimer()
			resultC, err := gogrep.New(gogrep.WithThreads(threads)).Grep(context.TODO(), "[cf].+sh", data)
			if err != nil {
				b.Fatal(err)
			}
			for range resultC {
			}
		})
	}
}
