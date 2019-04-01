package templates

import (
	"math/rand"
	"strings"
	"text/template"
	"time"

	// 3rd Party
	"github.com/gofrs/uuid"
	"github.com/icrowley/fake"
	"github.com/jmcvetta/randutil"
	log "github.com/sirupsen/logrus"
)

// ParseTemplates parse all templates.
// This will parse all files specified in `directory` with the extension
// .template and return a `template.Template`
func ParseTemplates(directory string) (*template.Template, error) {
	fmap := template.FuncMap{
		"add":              add,
		"date":             getDate,
		"seq":              seq,
		"weightedSequence": weightedSequence,
		"uuid":             uniqueID,
		"company":          fakeCompany,
		"product":          fakeProduct,
		"city":             fakeCity,
		"state":            fakeState,
		"street":           fakeStreet,
		"zipCode":          fakeZip,
		"randomInt":        randomInteger,
		"description":      fakeDescription,
	}
	templ, err := template.New("").Funcs(fmap).ParseGlob(directory + "/*.template")
	if err != nil {
		return nil, err
	}
	return templ, nil
}

// RenderTemplate render a named template.
// Render the template `name` and return it as a string
func RenderTemplate(name string, tpl *template.Template) (string, error) {
	var buf strings.Builder
	err := tpl.ExecuteTemplate(&buf, name, "foobar")
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// weightedSequence return a sequence range with a weighted value
func weightedSequence() []int {
	choices := make([]randutil.Choice, 0, 6)
	choices = append(choices, randutil.Choice{20, 1})
	choices = append(choices, randutil.Choice{15, 3})
	choices = append(choices, randutil.Choice{8, 10})
	choices = append(choices, randutil.Choice{5, 100})
	choices = append(choices, randutil.Choice{3, 1000})
	choices = append(choices, randutil.Choice{1, 10000})

	result, err := randutil.WeightedChoice(choices)
	if err != nil {
		log.Fatal(err)
	}

	return make([]int, result.Item.(int))
}

// seq return a slice of size `size`
// useful for creating a for loop of size `size`
// {{range seq 100}}text{{end}}
func seq(size int) []int {
	return make([]int, size)
}

// add simple addition in a template
func add(augend int, addend int) int {
	return augend + addend
}

// uuid return a unique identifier as a string
func uniqueID() string {
	id, err := uuid.NewV4()
	if err != nil {
		log.WithFields(log.Fields{
			"message": err,
		}).Error("Could not generate a new uuid")
	}
	return id.String()
}

// fakeProduct generate a random product name and return as a string
func fakeProduct() string {
	return fake.Product()
}

// fakeCity generate a random city name and return as a string
func fakeCity() string {
	return fake.City()
}

// fakeState generate a random state and return as a string
func fakeState() string {
	return fake.State()
}

// fakeZip generate a random zip code and return as a string
func fakeZip() string {
	return fake.Zip()
}

// fakeStreet generate a random street and return as a string
func fakeStreet() string {
	return fake.StreetAddress()
}

// fakeCompany generate a random company name and return as a string
func fakeCompany() string {
	return fake.Brand()
}

// randomInteger generate a random integer between 1 and n.
// return as an int
func randomInteger(n int) int {
	return rand.Intn(n-1) + 1
}

// fakeDescription generate a random sentence and return as a string
func fakeDescription() string {
	return fake.Sentence()
}

// getDate return the current date and time
func getDate(format string) string {
	t := time.Now()
	return t.Format(format)
}
