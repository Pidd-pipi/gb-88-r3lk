package util

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v6"
)

// Token rendering for gofakeit-based dynamic responses.
// Example: {"name":"{{name.fullName}}","email":"{{internet.email}}"}
// Repeat syntax: {{repeat 5|{"id":{{number.int 1 999}},"name":"{{name.fullName}}"}}}

var (
	tokenRe = regexp.MustCompile(`\{\{([^}]+)\}\}`)
)

// RenderTemplate replaces {{...}} tokens in a template string with gofakeit values.
func RenderTemplate(tpl string) (string, error) {
	out := expandRepeats(tpl)
	return renderTokens(out)
}

// expandRepeats expands {{repeat N}} ... {{endrepeat}} blocks. The inner
// template may itself contain tokens such as {{number.int 1 999}}; the
// end marker is a separate unambiguous token.
func expandRepeats(input string) string {
	const startMarker = "{{repeat"
	const endMarker = "{{endrepeat}}"
	var sb strings.Builder
	i := 0
	for i < len(input) {
		if strings.HasPrefix(input[i:], startMarker) {
			rest := input[i+len(startMarker):]
			j := 0
			for j < len(rest) && (rest[j] == ' ' || rest[j] == '\t') {
				j++
			}
			start := j
			for j < len(rest) && rest[j] >= '0' && rest[j] <= '9' {
				j++
			}
			numStr := rest[start:j]
			if numStr != "" && j < len(rest) && rest[j] == '}' && (j+1 < len(rest)) && rest[j+1] == '}' {
				n, err := strconv.Atoi(numStr)
				bodyStart := i + len(startMarker) + j + 2
				endIdx := strings.Index(input[bodyStart:], endMarker)
				if err == nil && n > 0 && n <= 100 && endIdx != -1 {
					inner := input[bodyStart : bodyStart+endIdx]
					items := make([]string, 0, n)
					for t := 0; t < n; t++ {
						rendered, rerr := renderTokens(inner)
						if rerr != nil {
							break
						}
						items = append(items, rendered)
					}
					sb.WriteString(strings.Join(items, ","))
					i = bodyStart + endIdx + len(endMarker)
					continue
				}
			}
			sb.WriteString(startMarker)
			i += len(startMarker)
			continue
		}
		sb.WriteByte(input[i])
		i++
	}
	return sb.String()
}

func renderTokens(input string) (string, error) {
	return tokenRe.ReplaceAllStringFunc(input, func(m string) string {
		body := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(m, "{{"), "}}"))
		return renderToken(body)
	}), nil
}

func renderToken(token string) string {
	fields := strings.Fields(token)
	if len(fields) == 0 {
		return ""
	}
	name := fields[0]
	args := fields[1:]

	atoi := func(def int) int {
		if len(args) > 0 {
			if v, err := strconv.Atoi(args[0]); err == nil {
				return v
			}
		}
		return def
	}

	switch name {
	case "name.fullName":
		return gofakeit.Name()
	case "name.firstName":
		return gofakeit.FirstName()
	case "name.lastName":
		return gofakeit.LastName()
	case "name.prefix":
		return gofakeit.NamePrefix()
	case "internet.email":
		return gofakeit.Email()
	case "internet.userName":
		return gofakeit.Username()
	case "internet.password":
		return gofakeit.Password(true, true, true, true, false, 12)
	case "internet.url":
		return gofakeit.URL()
	case "internet.ipv4":
		return gofakeit.IPv4Address()
	case "internet.domain":
		return gofakeit.DomainName()
	case "phone.phone":
		return gofakeit.Phone()
	case "phone.cell":
		return gofakeit.PhoneFormatted()
	case "address.city":
		return gofakeit.City()
	case "address.street":
		return gofakeit.Street()
	case "address.zip":
		return gofakeit.Zip()
	case "address.country":
		return gofakeit.Country()
	case "company.name":
		return gofakeit.Company()
	case "company.suffix":
		return gofakeit.CompanySuffix()
	case "job.title":
		return gofakeit.JobTitle()
	case "job.company":
		return gofakeit.Company()
	case "date.date":
		return gofakeit.Date().Format("2006-01-02")
	case "date.timestamp":
		return gofakeit.Date().Format(time.RFC3339)
	case "date.birthday":
		return gofakeit.DateRange(time.Now().AddDate(-60, 0, 0), time.Now().AddDate(-18, 0, 0)).Format("2006-01-02")
	case "number.int":
		min, max := 1, 9999
		if len(args) >= 2 {
			min, _ = strconv.Atoi(args[0])
			max, _ = strconv.Atoi(args[1])
		} else if len(args) == 1 {
			max, _ = strconv.Atoi(args[0])
		}
		if min > max {
			min, max = max, min
		}
		return strconv.Itoa(gofakeit.Number(min, max))
	case "number.float":
		return strconv.FormatFloat(gofakeit.Float64Range(0, 1000), 'f', 2, 64)
	case "number.bool":
		return strconv.FormatBool(gofakeit.Bool())
	case "uuid.uuid":
		return gofakeit.UUID()
	case "lorem.word":
		return gofakeit.Word()
	case "lorem.sentence":
		n := 8
		if len(args) > 0 {
			n = atoi(8)
		}
		return gofakeit.Sentence(n)
	case "lorem.paragraph":
		return gofakeit.Paragraph(3, 3, 12, " ")
	case "color.name":
		return gofakeit.Color()
	case "person.title":
		return gofakeit.JobTitle()
	case "path.id":
		if len(args) > 0 {
			return args[0]
		}
		return ""
	case "query.param":
		if len(args) > 0 {
			return args[0]
		}
		return ""
	default:
		return fmt.Sprintf("%s", gofakeit.Word())
	}
}
