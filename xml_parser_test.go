package parser

import (
	"log/slog"
	"os"
	"testing"
)

func TestXMLParserWithAttributes(t *testing.T) {
	xmlContent := `<user id="123" name="John Doe" active="true">
		<email type="primary">john@example.com</email>
		<email type="secondary">john.doe@work.com</email>
		<profile>
			<age>30</age>
			<city>New York</city>
		</profile>
	</user>`

	result, err := parseXMLToGeneric(xmlContent)
	if err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	// Check that we have the expected flattened structure

	// Check user attributes
	if result["user/id"] != "123" {
		t.Errorf("Expected user/id to be '123', got %v", result["user/id"])
	}

	if result["user/name"] != "John Doe" {
		t.Errorf("Expected user/name to be 'John Doe', got %v", result["user/name"])
	}

	if result["user/active"] != "true" {
		t.Errorf("Expected user/active to be 'true', got %v", result["user/active"])
	}

	// Check email attributes
	emailTypes := []string{"primary", "secondary"}
	for i, expectedType := range emailTypes {
		emailTypeKey := "user/email/type"
		emailType := result[emailTypeKey]

		// Handle array of email types
		if emailArray, ok := emailType.([]interface{}); ok {
			if i < len(emailArray) {
				if emailArray[i] != expectedType {
					t.Errorf("Expected email[%d] type to be '%s', got %v", i, expectedType, emailArray[i])
				}
			}
		} else if i == 0 {
			// Single email case
			if emailType != expectedType {
				t.Errorf("Expected first email type to be '%s', got %v", expectedType, emailType)
			}
		}
	}

	// Check email text content
	expectedEmails := []string{"john@example.com", "john.doe@work.com"}
	emailText := result["user/email"]
	if emailArray, ok := emailText.([]interface{}); ok {
		for i, expectedEmail := range expectedEmails {
			if i < len(emailArray) {
				if emailArray[i] != expectedEmail {
					t.Errorf("Expected email[%d] to be '%s', got %v", i, expectedEmail, emailArray[i])
				}
			}
		}
	}

	// Check profile content
	if result["user/profile/age"] != "30" {
		t.Errorf("Expected age to be '30', got %v", result["user/profile/age"])
	}

	if result["user/profile/city"] != "New York" {
		t.Errorf("Expected city to be 'New York', got %v", result["user/profile/city"])
	}

	// Check that user element exists as nested structure
	userMap, ok := result["user"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected user to be a map")
	}

	// Check nested structure contains expected elements
	if userMap["email"] == nil {
		t.Error("Expected user map to contain email element")
	}

	if userMap["profile"] == nil {
		t.Error("Expected user map to contain profile element")
	}

	t.Logf("XML parsing result: %+v", result)
}

func TestXMLHelperFunctions(t *testing.T) {
	xmlContent := `<product id="456" name="Widget" available="true">
		<price currency="USD">29.99</price>
		<description>A useful widget</description>
	</product>`

	result, err := parseXMLToGeneric(xmlContent)
	if err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	helper := XMLHelper{}

	// Test GetXMLAttribute for product attributes
	id := helper.GetXMLAttribute(result, "product", "id")
	if id != "456" {
		t.Errorf("Expected product id '456', got '%s'", id)
	}

	name := helper.GetXMLAttribute(result, "product", "name")
	if name != "Widget" {
		t.Errorf("Expected product name 'Widget', got '%s'", name)
	}

	// Test GetXMLAttribute for price currency
	currency := helper.GetXMLAttribute(result, "product/price", "currency")
	if currency != "USD" {
		t.Errorf("Expected price currency 'USD', got '%s'", currency)
	}

	// Test GetXMLText for elements with text content
	description := helper.GetXMLText(result, "product/description")
	if description != "A useful widget" {
		t.Errorf("Expected description 'A useful widget', got '%s'", description)
	}

	price := helper.GetXMLText(result, "product/price")
	if price != "29.99" {
		t.Errorf("Expected price '29.99', got '%s'", price)
	}

	// Test HasXMLAttribute
	if !helper.HasXMLAttribute(result, "product", "id") {
		t.Error("Expected product to have 'id' attribute")
	}

	if helper.HasXMLAttribute(result, "product", "nonexistent") {
		t.Error("Expected product not to have 'nonexistent' attribute")
	}

	// Test HasXMLElement
	if !helper.HasXMLElement(result, "product") {
		t.Error("Expected to have 'product' element")
	}

	if !helper.HasXMLElement(result, "product/price") {
		t.Error("Expected to have 'product/price' element")
	}

	if helper.HasXMLElement(result, "nonexistent") {
		t.Error("Expected not to have 'nonexistent' element")
	}

	// Test ListXMLAttributes for product
	attrs := helper.ListXMLAttributes(result, "product")
	expectedAttrs := []string{"id", "name", "available"}
	if len(attrs) != len(expectedAttrs) {
		t.Logf("Got attributes: %v", attrs)
		// Check if the basic expected attributes are present
		hasId := false
		hasName := false
		hasAvailable := false
		for _, attr := range attrs {
			switch attr {
			case "id":
				hasId = true
			case "name":
				hasName = true
			case "available":
				hasAvailable = true
			}
		}
		if !hasId || !hasName || !hasAvailable {
			t.Errorf("Expected product to have at least 'id', 'name', and 'available' attributes")
		}
	}

	// Test ListXMLElements
	elements := helper.ListXMLElements(result)
	if len(elements) < 1 { // Should have at least product
		t.Errorf("Expected at least 1 elements, got %d: %v", len(elements), elements)
	}

	t.Logf("XML parsing result: %+v", result)
}

func TestXMLParsingEdgeCases(t *testing.T) {
	// Test empty XML
	_, err := parseXMLToGeneric("")
	if err == nil {
		t.Error("Expected error for empty XML")
	}

	// Test whitespace only
	_, err = parseXMLToGeneric("   \n\t  ")
	if err == nil {
		t.Error("Expected error for whitespace-only XML")
	}

	// Test simple text node
	result, err := parseXMLToGeneric("<note>Simple note</note>")
	if err != nil {
		t.Fatalf("Failed to parse simple XML: %v", err)
	}

	// In our flattened structure, simple elements should be accessible directly by their path
	if result["note"] == nil {
		t.Error("Expected 'note' element to exist")
	}

	// The note value should be accessible as text from the nested structure
	noteMap, ok := result["note"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected note to be a map, got %T", result["note"])
	}

	if noteMap["note"] != "Simple note" {
		t.Errorf("Expected note text to be 'Simple note', got %v", noteMap["note"])
	}

	// Test self-closing tag
	result, err = parseXMLToGeneric(`<img src="test.jpg" alt="Test" />`)
	if err != nil {
		t.Fatalf("Failed to parse self-closing XML: %v", err)
	}

	if result["img/src"] != "test.jpg" {
		t.Errorf("Expected src 'test.jpg', got %v", result["img/src"])
	}

	if result["img/alt"] != "Test" {
		t.Errorf("Expected alt 'Test', got %v", result["img/alt"])
	}

	// Check that img element exists as empty in flattened structure
	if result["img"] == nil {
		t.Error("Expected 'img' element to exist")
	}
}

func TestXMLParsingWithDuplicateChildren(t *testing.T) {
	xmlContent := `<items>
		<item id="1">First</item>
		<item id="2">Second</item>
		<item id="3">Third</item>
	</items>`

	result, err := parseXMLToGeneric(xmlContent)
	if err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	// Check that we have arrays for duplicate items in flattened structure
	itemTexts := result["items/item"]
	itemArray, ok := itemTexts.([]interface{})
	if !ok {
		t.Fatalf("Expected items/item to be an array, got %T", itemTexts)
	}

	if len(itemArray) != 3 {
		t.Errorf("Expected 3 items, got %d", len(itemArray))
	}

	// Check item text values
	expectedTexts := []string{"First", "Second", "Third"}
	for i, expectedText := range expectedTexts {
		if i < len(itemArray) {
			if itemArray[i] != expectedText {
				t.Errorf("Expected item[%d] text to be '%s', got %v", i, expectedText, itemArray[i])
			}
		}
	}

	// Check item attributes array
	itemIds := result["items/item/id"]
	idArray, ok := itemIds.([]interface{})
	if !ok {
		t.Fatalf("Expected items/item/id to be an array, got %T", itemIds)
	}

	expectedIds := []string{"1", "2", "3"}
	for i, expectedId := range expectedIds {
		if i < len(idArray) {
			if idArray[i] != expectedId {
				t.Errorf("Expected item[%d] id to be '%s', got %v", i, expectedId, idArray[i])
			}
		}
	}

	// Check that items element exists as nested structure
	itemsMap, ok := result["items"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected items to be a map")
	}

	// Check nested structure contains item array
	if itemsMap["item"] == nil {
		t.Error("Expected items map to contain item element")
	}

	t.Logf("Successfully parsed XML with duplicate children: %+v", result)
}

func TestXMLParsingInvalidXML(t *testing.T) {
	// Test malformed XML
	_, err := parseXMLToGeneric("<invalid><unclosed>")
	if err == nil {
		t.Error("Expected error for malformed XML")
	}

	// Test invalid characters
	_, err = parseXMLToGeneric("<test>Invalid\x00character</test>")
	if err == nil {
		t.Error("Expected error for invalid characters")
	}
}

// TestFlattenedXMLStructure tests the new flattened XML structure format
func TestFlattenedXMLStructure(t *testing.T) {
	// Set debug logging
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	xmlContent := `<xml>
  <key attr1="1" attr2="2">
    <value1 attr="x"/>
    <value2 attr="y"/>
    <value3 attr="z">
      <child1 attr="c"/>
    </value3>
  </key>
  <tag attr1="a" attr2="b">tags</tag>
</xml>`

	result, err := parseXMLToGeneric(xmlContent)
	if err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	// Debug: print all keys in result
	t.Logf("XML parsing result keys: %v", result)
	for k, v := range result {
		t.Logf("Key: %s, Value: %v", k, v)
	}

	// Verify key element attributes
	if result["xml/key/attr1"] != "1" {
		t.Errorf("Expected xml/key/attr1 to be '1', got '%v'", result["xml/key/attr1"])
	}
	if result["xml/key/attr2"] != "2" {
		t.Errorf("Expected xml/key/attr2 to be '2', got '%v'", result["xml/key/attr2"])
	}

	// Verify tag element and attributes
	if result["xml/tag"] != "tags" {
		t.Errorf("Expected xml/tag to be 'tags', got '%v'", result["xml/tag"])
	}
	if result["xml/tag/attr1"] != "a" {
		t.Errorf("Expected xml/tag/attr1 to be 'a', got '%v'", result["xml/tag/attr1"])
	}
	if result["xml/tag/attr2"] != "b" {
		t.Errorf("Expected xml/tag/attr2 to be 'b', got '%v'", result["xml/tag/attr2"])
	}

	// Verify key has nested structure
	keyValue, keyExists := result["xml/key"]
	if !keyExists {
		t.Fatal("Expected 'xml/key' element to exist")
	}

	keyMap, isMap := keyValue.(map[string]interface{})
	if !isMap {
		t.Fatalf("Expected 'key' to be a map, got %T", keyValue)
	}

	// Check nested attributes
	if keyMap["value1/attr"] != "x" {
		t.Errorf("Expected value1/attr to be 'x', got '%v'", keyMap["value1/attr"])
	}
	if keyMap["value2/attr"] != "y" {
		t.Errorf("Expected value2/attr to be 'y', got '%v'", keyMap["value2/attr"])
	}
	if keyMap["value3/attr"] != "z" {
		t.Errorf("Expected value3/attr to be 'z', got '%v'", keyMap["value3/attr"])
	}

	// Check deeply nested structure
	value3, value3Exists := keyMap["value3"]
	if !value3Exists {
		t.Fatal("Expected 'value3' element to exist in key")
	}

	value3Map, isMap := value3.(map[string]interface{})
	if !isMap {
		t.Fatalf("Expected 'value3' to be a map, got %T", value3)
	}

	if value3Map["child1/attr"] != "c" {
		t.Errorf("Expected child1/attr to be 'c', got '%v'", value3Map["child1/attr"])
	}

	t.Logf("Successfully parsed XML structure: %+v", result)
}

// TestXMLArrayHandling tests that multiple children with same name become arrays
func TestXMLArrayHandling(t *testing.T) {
	xmlContent := `<xml>
  <items>
    <item id="1">First Item</item>
    <item id="2">Second Item</item>
    <item id="3">Third Item</item>
  </items>
  <category name="electronics">Electronics</category>
  <category name="books">Books</category>
</xml>`

	result, err := parseXMLToGeneric(xmlContent)
	if err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	// Verify items contains an array of item elements in flattened structure
	itemsValue, itemsExists := result["xml/items"]
	if !itemsExists {
		t.Fatal("Expected 'xml/items' element to exist")
	}

	itemsMap, isMap := itemsValue.(map[string]interface{})
	if !isMap {
		t.Fatalf("Expected 'xml/items' to be a map, got %T", itemsValue)
	}

	// Check that item is an array
	itemArray, itemExists := itemsMap["item"]
	if !itemExists {
		t.Fatal("Expected 'item' elements to exist in items")
	}

	itemSlice, isSlice := itemArray.([]interface{})
	if !isSlice {
		t.Fatalf("Expected 'item' to be an array, got %T", itemArray)
	}

	if len(itemSlice) != 3 {
		t.Errorf("Expected 3 item elements, got %d", len(itemSlice))
	}

	// Check item values using flattened paths
	itemTexts := result["xml/items/item"]
	itemTextSlice, isSlice := itemTexts.([]interface{})
	if !isSlice {
		t.Fatalf("Expected 'xml/items/item' to be an array, got %T", itemTexts)
	}

	expectedValues := []string{"First Item", "Second Item", "Third Item"}
	for i, expectedValue := range expectedValues {
		if i < len(itemTextSlice) {
			if itemTextSlice[i] != expectedValue {
				t.Errorf("Expected item[%d] to be '%s', got '%v'", i, expectedValue, itemTextSlice[i])
			}
		}
	}

	// Check that item attributes are also arrays
	itemIdArray, idExists := result["xml/items/item/id"]
	if !idExists {
		t.Fatal("Expected 'xml/items/item/id' attributes to exist")
	}

	idSlice, isSlice := itemIdArray.([]interface{})
	if !isSlice {
		t.Fatalf("Expected 'xml/items/item/id' to be an array, got %T", itemIdArray)
	}

	expectedIds := []string{"1", "2", "3"}
	for i, expectedId := range expectedIds {
		if i < len(idSlice) {
			if idSlice[i] != expectedId {
				t.Errorf("Expected item/id[%d] to be '%s', got '%v'", i, expectedId, idSlice[i])
			}
		}
	}

	// Check that category elements at root level are also arrays
	categoryArray, categoryExists := result["xml/category"]
	if !categoryExists {
		t.Fatal("Expected 'xml/category' elements to exist")
	}

	catSlice, isSlice := categoryArray.([]interface{})
	if !isSlice {
		t.Fatalf("Expected 'xml/category' to be an array, got %T", categoryArray)
	}

	if len(catSlice) != 2 {
		t.Errorf("Expected 2 category elements, got %d", len(catSlice))
	}

	// Check category attributes are arrays
	categoryNameArray, nameExists := result["xml/category/name"]
	if !nameExists {
		t.Fatal("Expected 'xml/category/name' attributes to exist")
	}

	nameSlice, isSlice := categoryNameArray.([]interface{})
	if !isSlice {
		t.Fatalf("Expected 'xml/category/name' to be an array, got %T", categoryNameArray)
	}

	expectedNames := []string{"electronics", "books"}
	for i, expectedName := range expectedNames {
		if i < len(nameSlice) {
			if nameSlice[i] != expectedName {
				t.Errorf("Expected category/name[%d] to be '%s', got '%v'", i, expectedName, nameSlice[i])
			}
		}
	}

	t.Logf("Successfully parsed XML with arrays: %+v", result)
}
