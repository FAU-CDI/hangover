package wisski2_test

import (
	"encoding/xml"
	"fmt"

	"github.com/FAU-CDI/hangover/internal/wisski2"
)

func ExamplePath() {
	var path wisski2.Path
	err := xml.Unmarshal([]byte(`
	<path>
		<id>publication</id>
		<weight>0</weight>
		<enabled>1</enabled>
		<group_id>0</group_id>
		<bundle>ba3c6c454f4ef7c9846e43524043d6f0</bundle>
		<field/>
		<fieldtype/>
		<displaywidget/>
		<formatterwidget/>
		<field_type_informative/>
		<cardinality>-1</cardinality>
		<path_array>
		<x>http://erlangen-crm.org/240307/E31_Document</x>
		</path_array>
		<datatype_property>empty</datatype_property>
		<short_name/>
		<disamb>0</disamb>
		<description/>
		<uuid>5c034251-92c5-4697-a4d1-c6cb969f60f5</uuid>
		<is_group>1</is_group>
		<name>Publication</name>
	</path>
	`), &path)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%#v", path)

	// Output: nope
}
