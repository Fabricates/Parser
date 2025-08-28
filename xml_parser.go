package parser

import (
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// parseXMLToGeneric parses XML content into a generic map structure for template use
// Returns a hierarchical structure where attributes are flattened with elementName/attributeName format
func parseXMLToGeneric(xmlContent string) (map[string]interface{}, error) {
	if strings.TrimSpace(xmlContent) == "" {
		slog.Debug("Empty XML content provided")
		return nil, fmt.Errorf("empty XML content")
	}

	// Parse XML into hierarchical format with flattened attributes
	parsedRoot, err := parseXMLHierarchical(xmlContent)
	if err != nil {
		slog.Debug("XML parsing failed", "error", err, "xml_length", len(xmlContent))
		return nil, err
	}

	slog.Debug("XML parsing successful", "root_parsed", parsedRoot != nil)
	return parsedRoot, nil
}

// parseXMLHierarchical parses XML into a hybrid structure with both flattened paths and nested maps
func parseXMLHierarchical(xmlContent string) (map[string]interface{}, error) {
	decoder := xml.NewDecoder(strings.NewReader(xmlContent))

	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				return nil, fmt.Errorf("no root element found")
			}
			return nil, err
		}

		switch t := token.(type) {
		case xml.StartElement:
			// Parse with both flattened and hierarchical structures
			result := make(map[string]interface{})
			nestedResult, err := parseXMLElementHybrid(decoder, t, "", result)
			if err != nil {
				return nil, err
			}

			// Add the root element as a nested structure
			result[t.Name.Local] = nestedResult

			// Add root element attributes at the top level for optimized structure
			for _, attr := range t.Attr {
				attrName := attr.Name.Local
				attrValue := attr.Value
				rootAttrKey := fmt.Sprintf("%s/%s", t.Name.Local, attrName)
				result[rootAttrKey] = attrValue
			}

			return result, nil
		}
	}
}

// parseXMLElementHybrid creates both flattened paths and nested map structures
func parseXMLElementHybrid(decoder *xml.Decoder, startElement xml.StartElement, parentPath string, flatResult map[string]interface{}) (map[string]interface{}, error) {
	elementName := startElement.Name.Local
	nestedResult := make(map[string]interface{})

	var currentPath string
	if parentPath == "" {
		currentPath = elementName
	} else {
		currentPath = parentPath + "/" + elementName
	}

	slog.Debug("Processing element hybrid", "name", elementName, "parentPath", parentPath, "currentPath", currentPath)

	// Add element attributes to both flattened and nested structures
	for _, attr := range startElement.Attr {
		attrName := attr.Name.Local
		attrValue := attr.Value

		// Flattened path: full path from root
		flatAttrKey := fmt.Sprintf("%s/%s", currentPath, attrName)
		slog.Debug("Adding flattened attribute", "key", flatAttrKey, "value", attrValue)

		// Handle multiple attributes with same name (arrays) in flattened structure
		if existingAttr, exists := flatResult[flatAttrKey]; exists {
			switch existingAttrArray := existingAttr.(type) {
			case []interface{}:
				flatResult[flatAttrKey] = append(existingAttrArray, attrValue)
			default:
				flatResult[flatAttrKey] = []interface{}{existingAttr, attrValue}
			}
		} else {
			flatResult[flatAttrKey] = attrValue
		}

		// Note: We don't add attributes to the element's own nested structure here
		// They will be added at the parent level by the parent element's processing
	}

	var textContent strings.Builder
	hasChildren := false

	for {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}

		switch t := token.(type) {
		case xml.StartElement:
			hasChildren = true
			childName := t.Name.Local

			// Parse child recursively
			childNested, err := parseXMLElementHybrid(decoder, t, currentPath, flatResult)
			if err != nil {
				return nil, err
			}

			// Add child to nested structure
			if existingChild, exists := nestedResult[childName]; exists {
				switch existingArray := existingChild.(type) {
				case []interface{}:
					nestedResult[childName] = append(existingArray, childNested)
				default:
					nestedResult[childName] = []interface{}{existingArray, childNested}
				}
			} else {
				nestedResult[childName] = childNested
			}

			// Add child's attributes to this parent level with childName/attrName format
			// This creates the optimized structure where attributes are at the same level as the element
			childStartElement := t // t is the xml.StartElement for the child
			for _, attr := range childStartElement.Attr {
				attrName := attr.Name.Local
				attrValue := attr.Value
				childAttrKey := fmt.Sprintf("%s/%s", childName, attrName)

				// Handle arrays for multiple children with same name and same attributes
				if existingAttr, exists := nestedResult[childAttrKey]; exists {
					switch existingAttrArray := existingAttr.(type) {
					case []interface{}:
						nestedResult[childAttrKey] = append(existingAttrArray, attrValue)
					default:
						nestedResult[childAttrKey] = []interface{}{existingAttr, attrValue}
					}
				} else {
					nestedResult[childAttrKey] = attrValue
				}
			}

		case xml.CharData:
			text := strings.TrimSpace(string(t))
			if text != "" {
				if textContent.Len() > 0 {
					textContent.WriteString(" ")
				}
				textContent.WriteString(text)
			}

		case xml.EndElement:
			if t.Name.Local == elementName {
				finalText := strings.TrimSpace(textContent.String())

				// Handle text content for both structures
				if !hasChildren && finalText != "" {
					// Text-only element
					slog.Debug("Storing text element", "path", currentPath, "text", finalText)

					// Store in flattened structure with array support
					if existingValue, exists := flatResult[currentPath]; exists {
						switch existingArray := existingValue.(type) {
						case []interface{}:
							flatResult[currentPath] = append(existingArray, finalText)
						default:
							flatResult[currentPath] = []interface{}{existingValue, finalText}
						}
					} else {
						flatResult[currentPath] = finalText
					}

					nestedResult[elementName] = finalText
				} else if finalText != "" {
					// Element with both children and text
					nestedResult["_text"] = finalText
				} else if !hasChildren {
					// Empty element
					if existingValue, exists := flatResult[currentPath]; exists {
						switch existingArray := existingValue.(type) {
						case []interface{}:
							flatResult[currentPath] = append(existingArray, "")
						default:
							flatResult[currentPath] = []interface{}{existingValue, ""}
						}
					} else {
						flatResult[currentPath] = ""
					}
				}

				// For elements with children, store the nested structure in flattened path
				if hasChildren {
					if existingValue, exists := flatResult[currentPath]; exists {
						switch existingArray := existingValue.(type) {
						case []interface{}:
							flatResult[currentPath] = append(existingArray, nestedResult)
						default:
							flatResult[currentPath] = []interface{}{existingValue, nestedResult}
						}
					} else {
						flatResult[currentPath] = nestedResult
					}
				}

				return nestedResult, nil
			}
		}
	}
}

// XMLHelper provides template functions for XML manipulation
type XMLHelper struct{}

// GetXMLAttribute extracts a specific attribute from an XML node map
// Usage: {{xmlAttr .BodyXML "key" "attr1"}} to get the 'attr1' attribute from 'key' element
// Works with format (key/attr)
func (h XMLHelper) GetXMLAttribute(xmlMap map[string]interface{}, elementName, attrName string) string {
	// Try new flattened format
	attrKey := fmt.Sprintf("%s/%s", elementName, attrName)
	if attr, exists := xmlMap[attrKey]; exists {
		switch attrVal := attr.(type) {
		case string:
			return attrVal
		case []interface{}:
			if len(attrVal) > 0 {
				if str, ok := attrVal[0].(string); ok {
					return str
				}
			}
		}
	}
	return ""
}

// GetXMLAttributeArray extracts all attribute values as an array
// Usage: {{xmlAttrArray .BodyXML "item" "id"}} to get all 'id' attributes from 'item' elements
func (h XMLHelper) GetXMLAttributeArray(xmlMap map[string]interface{}, elementName, attrName string) []string {
	attrKey := fmt.Sprintf("%s/%s", elementName, attrName)
	var result []string

	if attr, exists := xmlMap[attrKey]; exists {
		switch attrVal := attr.(type) {
		case string:
			result = append(result, attrVal)
		case []interface{}:
			for _, val := range attrVal {
				if str, ok := val.(string); ok {
					result = append(result, str)
				}
			}
		}
	}
	return result
}

// GetXMLValue extracts the value of an XML element
// Usage: {{xmlValue .BodyXML "key"}} to get the value of 'key' element
// For arrays: returns the first element
func (h XMLHelper) GetXMLValue(xmlMap map[string]interface{}, elementName string) interface{} {
	if value, exists := xmlMap[elementName]; exists {
		switch val := value.(type) {
		case []interface{}:
			if len(val) > 0 {
				return val[0]
			}
			return ""
		default:
			return val
		}
	}
	return ""
}

// GetXMLValueArray extracts all values of an XML element as an array
// Usage: {{xmlValueArray .BodyXML "item"}} to get all 'item' element values
func (h XMLHelper) GetXMLValueArray(xmlMap map[string]interface{}, elementName string) []interface{} {
	if value, exists := xmlMap[elementName]; exists {
		switch val := value.(type) {
		case []interface{}:
			return val
		default:
			return []interface{}{val}
		}
	}
	return []interface{}{}
}

// GetXMLText extracts text content from an XML element
func (h XMLHelper) GetXMLText(xmlMap map[string]interface{}, elementName string) string {
	// Try new flattened format
	if value := h.GetXMLValue(xmlMap, elementName); value != nil {
		if textStr, ok := value.(string); ok {
			return textStr
		}
	}
	return ""
}

// GetXMLTextArray extracts all text content from XML elements as string array
func (h XMLHelper) GetXMLTextArray(xmlMap map[string]interface{}, elementName string) []string {
	values := h.GetXMLValueArray(xmlMap, elementName)
	var result []string
	for _, val := range values {
		if str, ok := val.(string); ok {
			result = append(result, str)
		}
	}
	return result
}

// HasXMLAttribute checks if an XML element has a specific attribute
// Usage: {{hasXMLAttr .BodyXML "key" "attr1"}}
func (h XMLHelper) HasXMLAttribute(xmlMap map[string]interface{}, elementName, attrName string) bool {
	attrKey := fmt.Sprintf("%s/%s", elementName, attrName)
	_, exists := xmlMap[attrKey]
	return exists
}

// HasXMLElement checks if an XML element exists
// Usage: {{hasXMLElement .BodyXML "key"}}
func (h XMLHelper) HasXMLElement(xmlMap map[string]interface{}, elementName string) bool {
	_, exists := xmlMap[elementName]
	return exists
}

// IsXMLArray checks if an XML element is an array (has multiple values)
// Usage: {{isXMLArray .BodyXML "item"}}
func (h XMLHelper) IsXMLArray(xmlMap map[string]interface{}, elementName string) bool {
	if value, exists := xmlMap[elementName]; exists {
		_, isArray := value.([]interface{})
		return isArray
	}
	return false
}

// XMLArrayLength returns the length of an XML element array
// Usage: {{xmlArrayLen .BodyXML "item"}}
func (h XMLHelper) XMLArrayLength(xmlMap map[string]interface{}, elementName string) int {
	if value, exists := xmlMap[elementName]; exists {
		switch val := value.(type) {
		case []interface{}:
			return len(val)
		default:
			return 1 // Single element
		}
	}
	return 0
}

// ListXMLAttributes returns all attribute names for a specific element
// Usage: {{range xmlAttrs .BodyXML "key"}}{{.}}{{end}}
func (h XMLHelper) ListXMLAttributes(xmlMap map[string]interface{}, elementName string) []string {
	var attrs []string
	prefix := elementName + "/"
	for key := range xmlMap {
		if strings.HasPrefix(key, prefix) {
			attrName := strings.TrimPrefix(key, prefix)
			// Make sure it's a direct attribute, not a nested element
			if !strings.Contains(attrName, "/") {
				attrs = append(attrs, attrName)
			}
		}
	}
	return attrs
}

// ListXMLElements returns all element names from the XML map
// Usage: {{range xmlElements .BodyXML}}{{.}}{{end}}
func (h XMLHelper) ListXMLElements(xmlMap map[string]interface{}) []string {
	var elements []string
	for key := range xmlMap {
		// Skip attribute keys (those containing "/")
		if !strings.Contains(key, "/") {
			elements = append(elements, key)
		}
	}
	return elements
}
