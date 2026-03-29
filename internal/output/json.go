package output

import (
	"encoding/json"
	"fmt"

	"github.com/jtsilverman/probe/internal/reviewer"
)

func PrintJSON(review *reviewer.Review) {
	data, err := json.MarshalIndent(review, "", "  ")
	if err != nil {
		fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
		return
	}
	fmt.Println(string(data))
}
