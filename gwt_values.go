package gocronometer

// The following constants contain header values required for GWT requests. These values are found by inspecting a
// request from the web app. When the web app is updated these values can change. The values provided here are the
// default that will be used by the library if new values are not provided.
const (
	GWTContentType = "text/x-gwt-rpc; charset=UTF-8"
	GWTModuleBase  = "https://cronometer.com/cronometer/"
	// GWTPermutation is the X-GWT-Permutation header value. Captured live from
	// the web app on 2026-07-17 (previous value 2E2ADC7983FC786FB6AFF6B9B2FEA5AF
	// stopped matching after Cronometer's 2026-07-16 deploy).
	GWTPermutation = "6B907FFD872F5DEE50BB42A45CEFDDDD"

	// GWTHeader is what appears to be a hash value that is provided at the beginning of every GWT request. As it
	// changes with app updates it appears to be related to validating the version the requester is expecting.
	// Captured live 2026-07-17 (previous value DA19253CB693C806016C64754CF28FFF).
	//
	// To re-capture both values after a Cronometer deploy: open cronometer.com
	// in a browser with devtools (or Playwright), log in, and inspect any POST
	// to /cronometer/app — the request body's second |-separated field is
	// GWTHeader, and the x-gwt-permutation request header is GWTPermutation.
	// Both are also overridable at runtime via ClientOptions.
	GWTHeader = "8119D24F8CC7814B83B62DD87A7C62D8"
)

// The following are the GWT procedure calls as found from inspection of the app.
const (

	// GWTGenerateAuthToken will generate a GWT auth token. The only known use case is for accessing non GWT API calls
	// such as data export.
	// The first parameter in the string should be the sesnonce and the second is the users ID.
	GWTGenerateAuthToken = "7|0|8|https://cronometer.com/cronometer/|" + GWTHeader + "|com.cronometer.shared.rpc.CronometerService|generateAuthorizationToken" +
		"|java.lang.String/2004016611|I|com.cronometer.shared.user.AuthScope/2065601159|%s|1|2|3|4|4|5|6|6|7|8|%s|3600|7|2|"

	// GWTAuthenticate will authenticate with the GWT api. The sesnonce should be set in the cookies.
	GWTAuthenticate = "7|0|5|https://cronometer.com/cronometer/|" + GWTHeader + "|com.cronometer.shared.rpc.CronometerService|authenticate|java.lang.Integer/3438268394|1|2|3|4|1|5|5|-300|"

	// GWTLogout will log the session out.
	// The only parameter should be the sesnonce.
	GWTLogout = "7|0|6|https://cronometer.com/cronometer/|" + GWTHeader + "|com.cronometer.shared.rpc.CronometerService|logout|java.lang.String/2004016611|%s|1|2|3|4|1|5|6|"

	// GWTFindMyFoods lists all custom foods for the logged-in user.
	// Parameter: sesnonce, userID
	GWTFindMyFoods = "7|0|7|https://cronometer.com/cronometer/|" + GWTHeader + "|com.cronometer.shared.rpc.CronometerService|findMyFoods|java.lang.String/2004016611|I|%s|1|2|3|4|2|5|6|7|%s|"

	// GWTGetFood retrieves a single food's details including ingredients.
	// Parameter: sesnonce, foodID
	GWTGetFood = "7|0|7|https://cronometer.com/cronometer/|" + GWTHeader + "|com.cronometer.shared.rpc.CronometerService|getFood|java.lang.String/2004016611|I|%s|1|2|3|4|2|5|6|7|%d|"

	// GWTGetDayInfo retrieves a single day's diary — the servings (food log) the web app
	// loads via CronometerService.getDayInfo, replacing the rate-limited CSV /export path.
	// (The response also carries biometrics + exercise, which we ignore — Apple Health covers those.)
	//
	// Params (in order): sesnonce, day, month (1-based), year, userID.
	// Layout note: the "6" right after the sesnonce ref is the string-table ref to the Day type;
	// the Day object then serializes as day|month|year (verified against captured web requests,
	// e.g. 2026-06-28 -> ...|8|6|28|6|2026|<userID>|).
	GWTGetDayInfo = "7|0|8|https://cronometer.com/cronometer/|" + GWTHeader + "|com.cronometer.shared.rpc.CronometerService|getDayInfo" +
		"|java.lang.String/2004016611|com.cronometer.shared.entries.models.Day/782579793|I|%s|1|2|3|4|3|5|6|7|8|6|%d|%d|%d|%s|"

	// GWTGetAllFood retrieves multiple foods' details in batch.
	// The request body is dynamically built with a variable number of food IDs.
	// Format: 7|0|8|baseUrl|header|service|getAllFood|String|ArrayList|sesnonce|1|2|3|4|2|5|6|7|6|N|8|id1|8|id2|...|
	// where N is the count and each id is wrapped with the Integer type reference (8).
	GWTGetAllFoodPrefix = "7|0|8|https://cronometer.com/cronometer/|" + GWTHeader + "|com.cronometer.shared.rpc.CronometerService|getAllFood|java.lang.String/2004016611|java.util.ArrayList/4159755760|%s|java.lang.Integer/3438268394|1|2|3|4|2|5|6|7|6|"
)
