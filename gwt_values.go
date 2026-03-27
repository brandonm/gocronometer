package gocronometer

// The following constants contain header values required for GWT requests. These values are found by inspecting a
// request from the web app. When the web app is updated these values can change. The values provided here are the
// default that will be used by the library if new values are not provided.
const (
	GWTContentType = "text/x-gwt-rpc; charset=UTF-8"
	GWTModuleBase  = "https://cronometer.com/cronometer/"
	GWTPermutation = "2E2ADC7983FC786FB6AFF6B9B2FEA5AF"

	// GWTHeader is what appears to be a hash value that is provided at the beginning of every GWT request. As it
	// changes with app updates it appears to be related to validating the version the requester is expecting.
	GWTHeader = "DA19253CB693C806016C64754CF28FFF"
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

	// GWTGetAllFood retrieves multiple foods' details in batch.
	// The request body is dynamically built with a variable number of food IDs.
	// Format: 7|0|8|baseUrl|header|service|getAllFood|String|ArrayList|sesnonce|1|2|3|4|2|5|6|7|6|N|8|id1|8|id2|...|
	// where N is the count and each id is wrapped with the Integer type reference (8).
	GWTGetAllFoodPrefix = "7|0|8|https://cronometer.com/cronometer/|" + GWTHeader + "|com.cronometer.shared.rpc.CronometerService|getAllFood|java.lang.String/2004016611|java.util.ArrayList/4159755760|%s|java.lang.Integer/3438268394|1|2|3|4|2|5|6|7|6|"
)
