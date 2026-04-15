package avl

// IOParam describes a known IO element with its human-readable name, unit, and data type.
type IOParam struct {
	ID          int
	Name        string
	Unit        string
	Description string
}

// Registry maps IO element IDs to their human-readable parameter definitions.
// This covers the mandatory set from the specification plus extended Teltonika IDs.
var Registry = map[int]IOParam{
	// ── Core I/O ────────────────────────────────────────────────────────────
	1:   {1, "digital_input_1", "", "Digital Input 1 / Ignition"},
	2:   {2, "digital_input_2", "", "Digital Input 2"},
	3:   {3, "digital_input_3", "", "Digital Input 3"},
	4:   {4, "digital_input_4", "", "Digital Input 4"},
	9:   {9, "analog_input_1", "mV", "Analog Input 1"},
	10:  {10, "analog_input_2", "mV", "Analog Input 2"},
	11:  {11, "iccid_1", "", "ICCID 1"},
	12:  {12, "fuel_used_gps", "mL", "Fuel Used GPS"},
	13:  {13, "fuel_rate_gps", "mL/h", "Fuel Rate GPS"},
	17:  {17, "rssi", "dBm", "GSM RSSI"},
	21:  {21, "gsm_signal", "", "GSM Signal Strength"},
	66:  {66, "external_voltage", "mV", "External Power Voltage"},
	67:  {67, "battery_voltage", "mV", "Internal Battery Voltage"},
	68:  {68, "battery_current", "mA", "Battery Current"},
	69:  {69, "gnss_status", "", "GNSS Status (0=OFF,1=ON,2=Sleep,3=Fail)"},
	113: {113, "battery_level", "%", "Battery Level"},
	175: {175, "auto_geofence", "", "Auto Geofence"},
	179: {179, "digital_output_1", "", "Digital Output 1"},
	180: {180, "digital_output_2", "", "Digital Output 2"},

	// ── Driver / RFID ────────────────────────────────────────────────────────
	238: {238, "user_id", "", "User ID (iButton / RFID UID)"},
	239: {239, "ignition", "", "Ignition (0=OFF, 1=ON)"},
	240: {240, "movement", "", "Movement Sensor (0=no, 1=yes)"},
	241: {241, "active_gsm_operator", "", "Active GSM Operator (MCC+MNC)"},

	// ── Odometer / Trip ──────────────────────────────────────────────────────
	16:  {16, "total_odometer", "m", "Total Odometer"},
	199: {199, "trip_odometer", "m", "Trip Odometer"},

	// ── Harsh Driving Events ─────────────────────────────────────────────────
	246: {246, "towing_detection", "", "Towing Detection"},
	247: {247, "crash_detection", "", "Crash Detection"},
	248: {248, "crash_trace_data", "", "Crash Trace Data ID"},
	249: {249, "jagged_driving", "", "Jagged Driving Event"},
	250: {250, "harsh_acceleration", "", "Harsh Acceleration Event"},
	251: {251, "harsh_braking", "", "Harsh Braking Event"},
	252: {252, "harsh_cornering", "", "Harsh Cornering Event"},

	// ── OBD-II / CAN ─────────────────────────────────────────────────────────
	256: {256, "vin", "", "Vehicle Identification Number"},
	263: {263, "engine_rpm", "RPM", "Engine RPM"},
	264: {264, "coolant_temp", "°C", "Engine Coolant Temperature"},
	270: {270, "throttle_pos", "%", "Throttle Position"},
	271: {271, "engine_load", "%", "Calculated Engine Load"},
	274: {274, "fuel_level_obd", "%", "Fuel Level (OBD)"},

	// ── Fuel (Analog) ────────────────────────────────────────────────────────
	30:  {30, "fuel_level_1", "mV", "Fuel Level Analog Input 1"},
	31:  {31, "fuel_level_2", "mV", "Fuel Level Analog Input 2"},

	// ── Temperature ──────────────────────────────────────────────────────────
	72:  {72, "temperature_1", "°C×10", "Dallas Temperature 1"},
	73:  {73, "temperature_2", "°C×10", "Dallas Temperature 2"},
	74:  {74, "temperature_3", "°C×10", "Dallas Temperature 3"},
	75:  {75, "temperature_4", "°C×10", "Dallas Temperature 4"},

	// ── Bluetooth / TPMS ────────────────────────────────────────────────────
	303: {303, "bt_sensor_1_temperature", "°C×10", "BT Sensor 1 Temperature"},
	304: {304, "bt_sensor_1_humidity", "%×10", "BT Sensor 1 Humidity"},
	305: {305, "bt_sensor_2_temperature", "°C×10", "BT Sensor 2 Temperature"},
	306: {306, "bt_sensor_2_humidity", "%×10", "BT Sensor 2 Humidity"},
	314: {314, "tpms_tire1_pressure", "Pa", "TPMS Tire 1 Pressure"},
	315: {315, "tpms_tire2_pressure", "Pa", "TPMS Tire 2 Pressure"},
	316: {316, "tpms_tire3_pressure", "Pa", "TPMS Tire 3 Pressure"},
	317: {317, "tpms_tire4_pressure", "Pa", "TPMS Tire 4 Pressure"},

	// ── CAN Bus (FMC) ────────────────────────────────────────────────────────
	320: {320, "can_vehicle_speed", "km/h", "CAN Vehicle Speed"},
	321: {321, "can_accelerator_pedal", "%", "CAN Accelerator Pedal Position"},
	322: {322, "can_fuel_consumed", "L×10", "CAN Fuel Consumed"},
	323: {323, "can_fuel_level", "L×10", "CAN Fuel Level"},
	326: {326, "can_engine_temp", "°C", "CAN Engine Temperature"},
	327: {327, "can_engine_rpm", "RPM", "CAN Engine RPM"},
	332: {332, "can_total_milage", "km", "CAN Total Mileage"},
	333: {333, "can_total_fuel_used", "L×10", "CAN Total Fuel Used"},

	// ── SOS / Security ──────────────────────────────────────────────────────
	236: {236, "sos_event", "", "SOS / Panic Button"},
	237: {237, "asset_tamper", "", "Asset Tamper Event"},
}

// Lookup returns the IOParam for a given IO element ID (ok=false if unknown).
func Lookup(id int) (IOParam, bool) {
	p, ok := Registry[id]
	return p, ok
}
