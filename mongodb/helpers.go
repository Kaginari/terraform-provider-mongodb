package mongodb

import (
	"encoding/base64"
	"fmt"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"strings"
)

func validateDiagFunc(validateFunc func(interface{}, string) ([]string, []error)) schema.SchemaValidateDiagFunc {
	return func(i interface{}, path cty.Path) diag.Diagnostics {
		warnings, errs := validateFunc(i, fmt.Sprintf("%+v", path))
		var diags diag.Diagnostics
		for _, warning := range warnings {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  warning,
			})
		}
		for _, err := range errs {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  err.Error(),
			})
		}
		return diags
	}
}

func ParseId(id string, expectedParts int) ([]string, error) {
	result, errEncoding := base64.StdEncoding.DecodeString(id)
	if errEncoding != nil {
		return nil, fmt.Errorf("unexpected format of ID Error : %s", errEncoding)
	}
	parts := strings.SplitN(string(result), ".", expectedParts)
	if len(parts) != expectedParts {
		return nil, fmt.Errorf("unexpected format of ID (%s), expected attribute1.attributeN", id)
	}

	for _, part := range parts {
		if part == "" {
			return nil, fmt.Errorf("invalid ID format: %s", result)
		}
	}

	return parts, nil
}

func SetId(data *schema.ResourceData, parts []string) {
	var id string
	for _, part := range parts {
		if len(id) == 0 {
			id = part
		} else {
			id += "." + part
		}
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(id))
	data.SetId(encoded)
}
