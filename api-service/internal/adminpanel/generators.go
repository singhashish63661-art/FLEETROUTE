package adminpanel

import (
	adminctx "github.com/GoAdminGroup/go-admin/context"
	"github.com/GoAdminGroup/go-admin/modules/config"
	"github.com/GoAdminGroup/go-admin/modules/db"
	"github.com/GoAdminGroup/go-admin/plugins/admin/modules/table"
	"github.com/GoAdminGroup/go-admin/template/types"
	"github.com/GoAdminGroup/go-admin/template/types/form"
)

// Generators exposes project tables in go-admin.
func Generators() table.GeneratorList {
	return table.GeneratorList{
		"tenants":         getTenantsTable,
		"users":           getUsersTable,
		"devices":         getDevicesTable,
		"vehicles":        getVehiclesTable,
		"drivers":         getDriversTable,
		"geofences":       getGeofencesTable,
		"geofence-events": getGeofenceEventsTable,
		"trips":           getTripsTable,
		"alert-rules":     getAlertRulesTable,
		"alerts":          getAlertsTable,
		"avl-records":     getAVLRecordsTable,
	}
}

func getTenantsTable(ctx *adminctx.Context) (tenantTable table.Table) {
	tenantTable = table.NewDefaultTable(ctx, table.Config{
		Driver:     config.GetDatabases().GetDefault().Driver,
		CanAdd:     true,
		Editable:   true,
		Deletable:  false,
		Exportable: true,
		Connection: table.DefaultConnectionName,
		PrimaryKey: table.PrimaryKey{
			Type: db.Varchar,
			Name: "id",
		},
	})

	info := tenantTable.GetInfo()
	info.AddField("ID", "id", db.Varchar).FieldSortable()
	info.AddField("Name", "name", db.Varchar).FieldSortable()
	info.AddField("Slug", "slug", db.Varchar).FieldSortable()
	info.AddField("Plan", "plan", db.Varchar)
	info.AddField("Max Devices", "max_devices", db.Int)
	info.AddField("Active", "is_active", db.Bool)
	info.AddField("Created At", "created_at", db.Timestamp)
	info.SetTable("tenants").SetTitle("Tenants").SetDescription("Tenant management")

	formList := tenantTable.GetForm()
	formList.AddField("Name", "name", db.Varchar, form.Text).FieldMust()
	formList.AddField("Slug", "slug", db.Varchar, form.Text).FieldMust()
	formList.AddField("Plan", "plan", db.Varchar, form.SelectSingle).
		FieldOptions(types.FieldOptions{
			{Text: "Standard", Value: "standard"},
			{Text: "Enterprise", Value: "enterprise"},
		}).
		FieldDefault("standard")
	formList.AddField("Max Devices", "max_devices", db.Int, form.Number).FieldDefault("100")
	formList.AddField("Active", "is_active", db.Bool, form.Switch).
		FieldOptions(types.FieldOptions{
			{Value: "0"},
			{Value: "1"},
		}).
		FieldDefault("1")
	formList.SetTable("tenants").SetTitle("Tenants").SetDescription("Tenant management")

	return
}

func getUsersTable(ctx *adminctx.Context) (userTable table.Table) {
	userTable = table.NewDefaultTable(ctx, table.Config{
		Driver:     config.GetDatabases().GetDefault().Driver,
		CanAdd:     false,
		Editable:   true,
		Deletable:  false,
		Exportable: true,
		Connection: table.DefaultConnectionName,
		PrimaryKey: table.PrimaryKey{
			Type: db.Varchar,
			Name: "id",
		},
	})

	info := userTable.GetInfo()
	info.AddField("ID", "id", db.Varchar).FieldSortable()
	info.AddField("Tenant ID", "tenant_id", db.Varchar)
	info.AddField("Name", "name", db.Varchar).FieldSortable()
	info.AddField("Email", "email", db.Varchar).FieldSortable()
	info.AddField("Role", "role", db.Varchar)
	info.AddField("Phone", "phone", db.Varchar)
	info.AddField("Active", "is_active", db.Bool)
	info.AddField("Last Login", "last_login_at", db.Timestamp)
	info.AddField("Created At", "created_at", db.Timestamp)
	info.SetTable("users").SetTitle("Users").SetDescription("Application users")

	formList := userTable.GetForm()
	formList.AddField("Name", "name", db.Varchar, form.Text).FieldMust()
	formList.AddField("Email", "email", db.Varchar, form.Email).FieldMust()
	formList.AddField("Role", "role", db.Varchar, form.SelectSingle).
		FieldOptions(types.FieldOptions{
			{Text: "Super Admin", Value: "super_admin"},
			{Text: "Tenant Admin", Value: "tenant_admin"},
			{Text: "Fleet Manager", Value: "fleet_manager"},
			{Text: "Dispatcher", Value: "dispatcher"},
			{Text: "Driver", Value: "driver"},
		}).
		FieldDefault("dispatcher")
	formList.AddField("Phone", "phone", db.Varchar, form.Text)
	formList.AddField("Active", "is_active", db.Bool, form.Switch).
		FieldOptions(types.FieldOptions{
			{Value: "0"},
			{Value: "1"},
		}).
		FieldDefault("1")
	formList.SetTable("users").SetTitle("Users").SetDescription("Application users")

	return
}

func getDevicesTable(ctx *adminctx.Context) (deviceTable table.Table) {
	deviceTable = table.NewDefaultTable(ctx, table.Config{
		Driver:     config.GetDatabases().GetDefault().Driver,
		CanAdd:     true,
		Editable:   true,
		Deletable:  false,
		Exportable: true,
		Connection: table.DefaultConnectionName,
		PrimaryKey: table.PrimaryKey{Type: db.Varchar, Name: "id"},
	})

	info := deviceTable.GetInfo()
	info.AddField("ID", "id", db.Varchar).FieldSortable()
	info.AddField("Tenant ID", "tenant_id", db.Varchar)
	info.AddField("IMEI", "imei", db.Varchar).FieldSortable()
	info.AddField("Name", "name", db.Varchar).FieldSortable()
	info.AddField("Protocol", "protocol", db.Varchar)
	info.AddField("Vehicle ID", "vehicle_id", db.Varchar)
	info.AddField("Last Seen", "last_seen_at", db.Timestamp)
	info.AddField("Created At", "created_at", db.Timestamp)
	info.SetTable("devices").SetTitle("Devices").SetDescription("Fleet GPS devices")

	formList := deviceTable.GetForm()
	formList.AddField("Tenant ID", "tenant_id", db.Varchar, form.Text).FieldMust()
	formList.AddField("IMEI", "imei", db.Varchar, form.Text).FieldMust()
	formList.AddField("Name", "name", db.Varchar, form.Text).FieldMust()
	formList.AddField("Protocol", "protocol", db.Varchar, form.SelectSingle).
		FieldOptions(types.FieldOptions{
			{Text: "Teltonika", Value: "teltonika"},
			{Text: "GT06", Value: "gt06"},
			{Text: "TK103", Value: "tk103"},
			{Text: "JT808", Value: "jt808"},
			{Text: "AIS140", Value: "ais140"},
		})
	formList.AddField("Vehicle ID", "vehicle_id", db.Varchar, form.Text)
	formList.SetTable("devices").SetTitle("Devices").SetDescription("Fleet GPS devices")
	return
}

func getVehiclesTable(ctx *adminctx.Context) (vehicleTable table.Table) {
	vehicleTable = table.NewDefaultTable(ctx, table.Config{
		Driver:     config.GetDatabases().GetDefault().Driver,
		CanAdd:     true,
		Editable:   true,
		Deletable:  false,
		Exportable: true,
		Connection: table.DefaultConnectionName,
		PrimaryKey: table.PrimaryKey{Type: db.Varchar, Name: "id"},
	})

	info := vehicleTable.GetInfo()
	info.AddField("ID", "id", db.Varchar).FieldSortable()
	info.AddField("Tenant ID", "tenant_id", db.Varchar)
	info.AddField("Registration", "registration", db.Varchar).FieldSortable()
	info.AddField("Make", "make", db.Varchar)
	info.AddField("Model", "model", db.Varchar)
	info.AddField("Year", "year", db.Int)
	info.AddField("Fuel Type", "fuel_type", db.Varchar)
	info.AddField("Device ID", "device_id", db.Varchar)
	info.AddField("Driver ID", "driver_id", db.Varchar)
	info.AddField("Created At", "created_at", db.Timestamp)
	info.SetTable("vehicles").SetTitle("Vehicles").SetDescription("Fleet vehicles")

	formList := vehicleTable.GetForm()
	formList.AddField("Tenant ID", "tenant_id", db.Varchar, form.Text).FieldMust()
	formList.AddField("Registration", "registration", db.Varchar, form.Text).FieldMust()
	formList.AddField("Make", "make", db.Varchar, form.Text)
	formList.AddField("Model", "model", db.Varchar, form.Text)
	formList.AddField("Year", "year", db.Int, form.Number)
	formList.AddField("Fuel Type", "fuel_type", db.Varchar, form.Text)
	formList.AddField("Device ID", "device_id", db.Varchar, form.Text)
	formList.AddField("Driver ID", "driver_id", db.Varchar, form.Text)
	formList.SetTable("vehicles").SetTitle("Vehicles").SetDescription("Fleet vehicles")
	return
}

func getDriversTable(ctx *adminctx.Context) (driverTable table.Table) {
	driverTable = table.NewDefaultTable(ctx, table.Config{
		Driver:     config.GetDatabases().GetDefault().Driver,
		CanAdd:     true,
		Editable:   true,
		Deletable:  false,
		Exportable: true,
		Connection: table.DefaultConnectionName,
		PrimaryKey: table.PrimaryKey{Type: db.Varchar, Name: "id"},
	})

	info := driverTable.GetInfo()
	info.AddField("ID", "id", db.Varchar).FieldSortable()
	info.AddField("Tenant ID", "tenant_id", db.Varchar)
	info.AddField("Name", "name", db.Varchar).FieldSortable()
	info.AddField("License Number", "license_number", db.Varchar)
	info.AddField("RFID UID", "rfid_uid", db.Varchar)
	info.AddField("Phone", "phone", db.Varchar)
	info.AddField("Email", "email", db.Varchar)
	info.AddField("Score", "score_current", db.Float)
	info.AddField("Created At", "created_at", db.Timestamp)
	info.SetTable("drivers").SetTitle("Drivers").SetDescription("Fleet drivers")

	formList := driverTable.GetForm()
	formList.AddField("Tenant ID", "tenant_id", db.Varchar, form.Text).FieldMust()
	formList.AddField("Name", "name", db.Varchar, form.Text).FieldMust()
	formList.AddField("License Number", "license_number", db.Varchar, form.Text)
	formList.AddField("RFID UID", "rfid_uid", db.Varchar, form.Text)
	formList.AddField("Phone", "phone", db.Varchar, form.Text)
	formList.AddField("Email", "email", db.Varchar, form.Email)
	formList.SetTable("drivers").SetTitle("Drivers").SetDescription("Fleet drivers")
	return
}

func getGeofencesTable(ctx *adminctx.Context) (geofenceTable table.Table) {
	geofenceTable = table.NewDefaultTable(ctx, table.Config{
		Driver:     config.GetDatabases().GetDefault().Driver,
		CanAdd:     false,
		Editable:   true,
		Deletable:  false,
		Exportable: true,
		Connection: table.DefaultConnectionName,
		PrimaryKey: table.PrimaryKey{Type: db.Varchar, Name: "id"},
	})

	info := geofenceTable.GetInfo()
	info.AddField("ID", "id", db.Varchar).FieldSortable()
	info.AddField("Tenant ID", "tenant_id", db.Varchar)
	info.AddField("Name", "name", db.Varchar).FieldSortable()
	info.AddField("Description", "description", db.Text)
	info.AddField("Shape", "shape_type", db.Varchar)
	info.AddField("Radius (m)", "radius_m", db.Float)
	info.AddField("Color", "color", db.Varchar)
	info.AddField("Active", "is_active", db.Bool)
	info.AddField("Created At", "created_at", db.Timestamp)
	info.SetTable("geofences").SetTitle("Geofences").SetDescription("Geofence master data")

	formList := geofenceTable.GetForm()
	formList.AddField("Name", "name", db.Varchar, form.Text).FieldMust()
	formList.AddField("Description", "description", db.Text, form.TextArea)
	formList.AddField("Shape", "shape_type", db.Varchar, form.SelectSingle).
		FieldOptions(types.FieldOptions{
			{Text: "Circle", Value: "circle"},
			{Text: "Polygon", Value: "polygon"},
			{Text: "Corridor", Value: "corridor"},
		})
	formList.AddField("Radius (m)", "radius_m", db.Float, form.Number)
	formList.AddField("Color", "color", db.Varchar, form.Text)
	formList.AddField("Active", "is_active", db.Bool, form.Switch).
		FieldOptions(types.FieldOptions{{Value: "0"}, {Value: "1"}}).
		FieldDefault("1")
	formList.SetTable("geofences").SetTitle("Geofences").SetDescription("Geofence master data")
	return
}

func getGeofenceEventsTable(ctx *adminctx.Context) (eventsTable table.Table) {
	eventsTable = table.NewDefaultTable(ctx, table.Config{
		Driver:     config.GetDatabases().GetDefault().Driver,
		CanAdd:     false,
		Editable:   false,
		Deletable:  false,
		Exportable: true,
		Connection: table.DefaultConnectionName,
		PrimaryKey: table.PrimaryKey{Type: db.Varchar, Name: "id"},
	})

	info := eventsTable.GetInfo()
	info.AddField("ID", "id", db.Varchar).FieldSortable()
	info.AddField("Tenant ID", "tenant_id", db.Varchar)
	info.AddField("Geofence ID", "geofence_id", db.Varchar)
	info.AddField("Device ID", "device_id", db.Varchar)
	info.AddField("Vehicle ID", "vehicle_id", db.Varchar)
	info.AddField("Event Type", "event_type", db.Varchar)
	info.AddField("Triggered At", "triggered_at", db.Timestamp).FieldSortable()
	info.AddField("Duration (s)", "duration_s", db.Int)
	info.AddField("Speed", "speed", db.Int)
	info.SetTable("geofence_events").SetTitle("Geofence Events").SetDescription("Entry and exit events")
	return
}

func getTripsTable(ctx *adminctx.Context) (tripsTable table.Table) {
	tripsTable = table.NewDefaultTable(ctx, table.Config{
		Driver:     config.GetDatabases().GetDefault().Driver,
		CanAdd:     false,
		Editable:   false,
		Deletable:  false,
		Exportable: true,
		Connection: table.DefaultConnectionName,
		PrimaryKey: table.PrimaryKey{Type: db.Varchar, Name: "id"},
	})

	info := tripsTable.GetInfo()
	info.AddField("ID", "id", db.Varchar).FieldSortable()
	info.AddField("Tenant ID", "tenant_id", db.Varchar)
	info.AddField("Device ID", "device_id", db.Varchar)
	info.AddField("Vehicle ID", "vehicle_id", db.Varchar)
	info.AddField("Driver ID", "driver_id", db.Varchar)
	info.AddField("Started At", "started_at", db.Timestamp).FieldSortable()
	info.AddField("Ended At", "ended_at", db.Timestamp)
	info.AddField("Distance (m)", "distance_m", db.Int)
	info.AddField("Duration (s)", "duration_s", db.Int)
	info.AddField("Max Speed", "max_speed", db.Int)
	info.AddField("Avg Speed", "avg_speed", db.Float)
	info.AddField("Idle Time (s)", "idle_time_s", db.Int)
	info.SetTable("trips").SetTitle("Trips").SetDescription("Trip analytics")
	return
}

func getAlertRulesTable(ctx *adminctx.Context) (rulesTable table.Table) {
	rulesTable = table.NewDefaultTable(ctx, table.Config{
		Driver:     config.GetDatabases().GetDefault().Driver,
		CanAdd:     true,
		Editable:   true,
		Deletable:  false,
		Exportable: true,
		Connection: table.DefaultConnectionName,
		PrimaryKey: table.PrimaryKey{Type: db.Varchar, Name: "id"},
	})

	info := rulesTable.GetInfo()
	info.AddField("ID", "id", db.Varchar).FieldSortable()
	info.AddField("Tenant ID", "tenant_id", db.Varchar)
	info.AddField("Name", "name", db.Varchar).FieldSortable()
	info.AddField("Template", "template_id", db.Varchar)
	info.AddField("Cooldown (s)", "cooldown_s", db.Int)
	info.AddField("Active", "is_active", db.Bool)
	info.AddField("Created At", "created_at", db.Timestamp)
	info.SetTable("alert_rules").SetTitle("Alert Rules").SetDescription("Alert rule engine configuration")

	formList := rulesTable.GetForm()
	formList.AddField("Tenant ID", "tenant_id", db.Varchar, form.Text).FieldMust()
	formList.AddField("Name", "name", db.Varchar, form.Text).FieldMust()
	formList.AddField("Template", "template_id", db.Varchar, form.Text)
	formList.AddField("Conditions JSON", "conditions", db.Text, form.TextArea).FieldMust()
	formList.AddField("Actions JSON", "actions", db.Text, form.TextArea).FieldMust()
	formList.AddField("Cooldown (s)", "cooldown_s", db.Int, form.Number).FieldDefault("300")
	formList.AddField("Active", "is_active", db.Bool, form.Switch).
		FieldOptions(types.FieldOptions{{Value: "0"}, {Value: "1"}}).
		FieldDefault("1")
	formList.SetTable("alert_rules").SetTitle("Alert Rules").SetDescription("Alert rule engine configuration")
	return
}

func getAlertsTable(ctx *adminctx.Context) (alertsTable table.Table) {
	alertsTable = table.NewDefaultTable(ctx, table.Config{
		Driver:     config.GetDatabases().GetDefault().Driver,
		CanAdd:     false,
		Editable:   false,
		Deletable:  false,
		Exportable: true,
		Connection: table.DefaultConnectionName,
		PrimaryKey: table.PrimaryKey{Type: db.Varchar, Name: "id"},
	})

	info := alertsTable.GetInfo()
	info.AddField("ID", "id", db.Varchar).FieldSortable()
	info.AddField("Tenant ID", "tenant_id", db.Varchar)
	info.AddField("Device ID", "device_id", db.Varchar)
	info.AddField("Rule ID", "rule_id", db.Varchar)
	info.AddField("Type", "alert_type", db.Varchar)
	info.AddField("Severity", "severity", db.Varchar)
	info.AddField("Message", "message", db.Text)
	info.AddField("Triggered At", "triggered_at", db.Timestamp).FieldSortable()
	info.AddField("Acknowledged At", "acknowledged_at", db.Timestamp)
	info.AddField("Acknowledged By", "acknowledged_by", db.Varchar)
	info.SetTable("alerts").SetTitle("Alerts").SetDescription("Triggered alerts")
	return
}

func getAVLRecordsTable(ctx *adminctx.Context) (avlTable table.Table) {
	avlTable = table.NewDefaultTable(ctx, table.Config{
		Driver:     config.GetDatabases().GetDefault().Driver,
		CanAdd:     false,
		Editable:   false,
		Deletable:  false,
		Exportable: true,
		Connection: table.DefaultConnectionName,
		PrimaryKey: table.PrimaryKey{Type: db.Varchar, Name: "device_id"},
	})

	info := avlTable.GetInfo()
	info.AddField("Device ID", "device_id", db.Varchar).FieldSortable()
	info.AddField("Tenant ID", "tenant_id", db.Varchar)
	info.AddField("Timestamp", "timestamp", db.Timestamp).FieldSortable()
	info.AddField("Lat", "lat", db.Float)
	info.AddField("Lng", "lng", db.Float)
	info.AddField("Speed", "speed", db.Int)
	info.AddField("Heading", "heading", db.Int)
	info.AddField("Ignition", "ignition", db.Bool)
	info.AddField("Movement", "movement", db.Bool)
	info.AddField("SOS", "sos_event", db.Bool)
	info.SetTable("avl_records").SetTitle("AVL Records").SetDescription("Raw telemetry stream")
	return
}
