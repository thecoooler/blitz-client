package client

import (
	"maps"
	"reflect"
	"testing"
)

func TestCatalogReadsRegularItemsWithLocalization(t *testing.T) {
	f := newFixture(t)
	f.maps()
	f.write("Strings/ru.yaml", `"#artefacts:kit/name": "Ремкомплект"`+"\n"+`"ARMOR_PIERCING_FULL": "Бронебойный снаряд"`+"\n")
	f.writeDVPL(commonItemDefs+"consumables/common.xml", "<root><nextAvailableId>9</nextAvailableId>"+
		"<repairKit><id>3</id><userString>#artefacts:kit/name</userString><description>#artefacts:kit/descr</description>"+
		"<icon>largeRepairkit</icon><script>Repair<automatic>true</automatic></script></repairKit>"+
		"<hiddenAbility><id>4</id><icon>ability</icon></hiddenAbility></root>")
	f.write(commonItemDefs+"consumables/madgames.xml", "<root><eventKit><id>5</id><userString>x</userString></eventKit></root>")
	f.write(commonItemDefs+"optional_devices.xml", "<root><rammer><id>100</id><userString>rammer</userString></rammer></root>")
	f.write("available_nations.yaml", "available_nations:\n  - ussr\n  - usa\n  - china\n")
	f.write("XML/item_defs/vehicles/ussr/components/shells.xml", "<root><icons>"+
		"<ap>ARMOR_PIERCING.png 0 0</ap><ap_premium>ARMOR_PIERCING_PREMIUM.png 0 0</ap_premium></icons>"+
		"<_76mm_BR><id>2</id><icon>ap_premium</icon><kind>ARMOR_PIERCING</kind></_76mm_BR></root>")
	f.writeDVPL("XML/item_defs/vehicles/usa/components/shells.xml", "<root><icons><ap>ARMOR_PIERCING.png 0 0</ap>"+
		"<atgm_heat>ATGM_HOLLOW_CHARGE.png 0 0</atgm_heat></icons></root>")

	catalog, err := DiscoverClientCatalog(f.data)
	if err != nil {
		t.Fatal(err)
	}
	want := &ClientItems{
		Shells: []ClientShellType{
			{ID: "ap", Name: "ARMOR_PIERCING", UserString: "Бронебойный снаряд"},
			{ID: "ap_premium", Name: "ARMOR_PIERCING_PREMIUM", Premium: true, UserString: "Бронебойный снаряд"},
			{ID: "atgm_heat", Name: "ATGM_HOLLOW_CHARGE"},
		},
		Consumables: []ClientItem{
			{ID: 3, Name: "repairKit", UserString: "Ремкомплект", Description: "#artefacts:kit/descr", IconKey: "largeRepairkit"},
			{ID: 4, Name: "hiddenAbility", IconKey: "ability"},
		},
		Provisions: []ClientItem{},
		Equipment:  []ClientItem{{ID: 100, Name: "rammer", UserString: "rammer"}},
	}
	if !reflect.DeepEqual(catalog.Items, want) {
		t.Fatalf("got %+v, want %+v", catalog.Items, want)
	}

	if keys := catalog.Items.IconKeys(ItemShells); !reflect.DeepEqual(keys, []string{"ap", "ap_premium", "atgm_heat"}) {
		t.Fatalf("shell icon keys %v", keys)
	}
	if keys := catalog.Items.IconKeys(ItemConsumables); !reflect.DeepEqual(keys, []string{"largeRepairkit", "ability"}) {
		t.Fatalf("consumable icon keys %v", keys)
	}

	path := f.write("XML/item_defs/vehicles/china/components/shells.xml", "<root><icons><ap>OTHER.png</ap></icons></root>")
	_, err = DiscoverClientCatalog(f.data)
	requireError(t, err, "CLIENT_XML_INVALID", path)
	f.write("XML/item_defs/vehicles/china/components/shells.xml", "<root><icons><ap/></icons></root>")
	_, err = DiscoverClientCatalog(f.data)
	requireError(t, err, "CLIENT_XML_INVALID", path)
	f.write("XML/item_defs/vehicles/china/components/shells.xml", "<root/>")
	itemsPath := f.write(commonItemDefs+"provisions/common.xml", "<root><bad><id>x</id></bad></root>")
	_, err = DiscoverClientCatalog(f.data)
	requireError(t, err, "CLIENT_XML_INVALID", itemsPath)
}

func TestReadsItemIconSpritesFromUIResources(t *testing.T) {
	f := newFixture(t)
	f.writeDVPL(supplyIconStyles[0].path, `
.equipment_item.kit * .equipment-img
{
    bg-sprite: "~res:/Gfx/Big/kit.txt";
}
.1st_april.equipment_item.kit * .equipment-img { bg-sprite: "~res:/Gfx/Event/kit.txt"; }
.equipment_item.food *   .equipment-img,
.equipment_item.other * .equipment-img { bg-sprite: "~res:/Gfx/Big/food.psd"; }
.equipment_item.nosprite * .equipment-img { size: 1; }`)
	f.write(supplyIconStyles[1].path, `.equipment_item.kit #icon { bg-sprite: "~res:/Gfx/Normal/kit"; }
.equipment_item.big-only ? #icon, .equipment_item.big-only #icon { bg-sprite: "~res:/Gfx/Big/only"; }`)
	f.write(supplyIconStyles[2].path, `.equipment_item.ability.ready #icon.small { bg-sprite: "~res:/Gfx/ready"; }
.equipment_item.small #icon.small { bg-sprite: "~res:/Gfx/small"; }
.equipment_item.small #icon { bg-sprite: "~res:/Gfx/regular"; }`)
	f.write(equipmentIconStyles[0].path, `.optional_device_item.rammer ? #Img, .optional_device_item.rammer #Img { bg-sprite: "~res:/Gfx/rammer.txt"; }`)
	f.write(shellIconMap, `- ["string", "iconPath", "when slot.iconType == \"ap\" -> \"~res:/Gfx/ammo/ap_l\", `+
		`when slot.iconType == \"r_he\" -> \"~res:/Gfx/ammo/rocket_he_l\", \"~res:/Gfx/ammo/ap_l\""]`)

	sprites, err := ReadItemIconSprites(f.data)
	if err != nil {
		t.Fatal(err)
	}
	supplies := map[string]string{
		"kit": "Gfx/Big/kit.txt", "food": "Gfx/Big/food.psd", "other": "Gfx/Big/food.psd",
		"big-only": "Gfx/Big/only", "small": "Gfx/regular",
	}
	for kind, want := range map[ItemKind]map[string]string{
		ItemConsumables: supplies, ItemProvisions: supplies,
		ItemEquipment: {"rammer": "Gfx/rammer.txt"},
		ItemShells:    {"ap": "Gfx/ammo/ap_l", "r_he": "Gfx/ammo/rocket_he_l"},
	} {
		if !maps.Equal(sprites[kind], want) {
			t.Errorf("%s: got %v, want %v", kind, sprites[kind], want)
		}
	}

	empty, err := ReadItemIconSprites(newFixture(t).data)
	if err != nil || len(empty[ItemConsumables]) != 0 || len(empty[ItemShells]) != 0 {
		t.Fatalf("got %v, %v", empty, err)
	}
}
