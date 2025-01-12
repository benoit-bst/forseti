package forseti

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"testing"
	"time"
	"io/ioutil"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeparturesApi(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	firstURI, err := url.Parse(fmt.Sprintf("file://%s/first.txt", fixtureDir))
	require.Nil(err)

	multipleURI, err := url.Parse(fmt.Sprintf("file://%s/multiple.txt", fixtureDir))
	require.Nil(err)

	var manager DataManager

	c, engine := gin.CreateTestContext(httptest.NewRecorder())
	engine = SetupRouter(&manager, engine)

	c.Request = httptest.NewRequest("GET", "/departures", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(400, w.Code)
	response := DeparturesResponse{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	assert.Nil(response.Departures)
	assert.NotEmpty(response.Message)

	c.Request = httptest.NewRequest("GET", "/departures?stop_id=3", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(503, w.Code)
	response = DeparturesResponse{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	assert.Nil(response.Departures)
	assert.NotEmpty(response.Message)

	//we load some data
	err = RefreshDepartures(&manager, *firstURI, defaultTimeout)
	assert.Nil(err)

	c.Request = httptest.NewRequest("GET", "/departures?stop_id=3", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)

	response = DeparturesResponse{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	assert.Empty(response.Message)
	require.NotNil(response.Departures)
	assert.NotEmpty(response.Departures)
	assert.Len(*response.Departures, 4)

	//these is no stop 5 in our dataset
	c.Request = httptest.NewRequest("GET", "/departures?stop_id=5", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)

	response = DeparturesResponse{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	assert.Empty(response.Message)
	require.NotNil(response.Departures)
	require.Empty(response.Departures)

	//load data with more than one stops
	err = RefreshDepartures(&manager, *multipleURI, defaultTimeout)
	assert.Nil(err)

	c.Request = httptest.NewRequest("GET", "/departures?stop_id=3&stop_id=4", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)

	response = DeparturesResponse{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	assert.Empty(response.Message)
	require.NotNil(response.Departures)
	assert.NotEmpty(response.Departures)
	assert.Len(*response.Departures, 8)

	c.Request = httptest.NewRequest("GET", "/departures?stop_id=3&stop_id=4&direction_type=both", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)

	response = DeparturesResponse{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	assert.Empty(response.Message)
	require.NotNil(response.Departures)
	assert.NotEmpty(response.Departures)
	assert.Len(*response.Departures, 8)

	c.Request = httptest.NewRequest("GET", "/departures?stop_id=3&stop_id=4&direction_type=forward", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)

	response = DeparturesResponse{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	assert.Empty(response.Message)
	require.NotNil(response.Departures)
	assert.NotEmpty(response.Departures)
	assert.Len(*response.Departures, 4)

	c.Request = httptest.NewRequest("GET", "/departures?stop_id=3&direction_type=backward", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)

	response = DeparturesResponse{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	assert.Empty(response.Message)
	require.NotNil(response.Departures)
	assert.NotEmpty(response.Departures)
	assert.Len(*response.Departures, 2)

	c.Request = httptest.NewRequest("GET", "/departures?stop_id=3&direction_type=aller", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(400, w.Code)
}

func TestStatusApiExist(t *testing.T) {
	require := require.New(t)
	var manager DataManager

	c, engine := gin.CreateTestContext(httptest.NewRecorder())
	SetupRouter(&manager, engine)

	c.Request = httptest.NewRequest("GET", "/status", nil)
	w := httptest.NewRecorder()
	require.Equal(200, w.Code)
}

func TestStatusApiHasLastUpdateTime(t *testing.T) {
	startTime := time.Now()
	assert := assert.New(t)
	require := require.New(t)
	firstURI, err := url.Parse(fmt.Sprintf("file://%s/first.txt", fixtureDir))
	require.Nil(err)

	var manager DataManager

	c, engine := gin.CreateTestContext(httptest.NewRecorder())
	engine = SetupRouter(&manager, engine)

	err = RefreshDepartures(&manager, *firstURI, defaultTimeout)
	assert.Nil(err)

	c.Request = httptest.NewRequest("GET", "/status", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)

	var response StatusResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	assert.Equal(response.Status, "ok")
	assert.True(response.LastDepartureUpdate.After(startTime))
	assert.True(response.LastDepartureUpdate.Before(time.Now()))
}

func TestStatusApiHasLastParkingUpdateTime(t *testing.T) {
	startTime := time.Now()
	assert := assert.New(t)
	require := require.New(t)
	parkingURI, err := url.Parse(fmt.Sprintf("file://%s/parkings.txt", fixtureDir))
	require.Nil(err)

	var manager DataManager

	c, engine := gin.CreateTestContext(httptest.NewRecorder())
	engine = SetupRouter(&manager, engine)

	err = RefreshParkings(&manager, *parkingURI, defaultTimeout)
	assert.Nil(err)

	c.Request = httptest.NewRequest("GET", "/status", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)

	var response StatusResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	assert.True(response.LastParkingUpdate.After(startTime))
	assert.True(response.LastParkingUpdate.Before(time.Now()))
}

func TestParkingsPRAPI(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	loc, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)
	updateTime, err := time.ParseInLocation("2006-01-02 15:04:05", "2018-09-17 19:29:00", loc)
	require.Nil(err)

	var manager DataManager
	manager.UpdateParkings(map[string]Parking{
		"riri":   {"Riri", "First of the name", updateTime, 1, 2, 3, 4},
		"fifi":   {"Fifi", "Second of the name", updateTime, 1, 2, 3, 4},
		"loulou": {"Loulou", "Third of the name", updateTime, 1, 2, 3, 4},
		"donald": {"Donald", "Donald THE Duck", updateTime, 1, 2, 3, 4},
	})

	c, engine := gin.CreateTestContext(httptest.NewRecorder())
	engine = SetupRouter(&manager, engine)

	c.Request = httptest.NewRequest("GET", "/parkings/P+R", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(http.StatusOK, w.Code)

	response := ParkingsResponse{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)

	parkings := response.Parkings
	sort.Sort(ByParkingResponseId(parkings))
	require.Len(parkings, 4)
	require.Len(response.Errors, 0)
	assert.Equal("Donald", parkings[0].ID)
	assert.Equal("Fifi", parkings[1].ID)
	assert.Equal("Loulou", parkings[2].ID)
	assert.Equal("Riri", parkings[3].ID)
}

func TestParkingsPRAPIwithParkingsID(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	loc, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)
	updateTime, err := time.ParseInLocation("2006-01-02 15:04:05", "2018-09-17 19:29:00", loc)
	require.Nil(err)

	var manager DataManager
	manager.UpdateParkings(map[string]Parking{
		"riri":   {"Riri", "First of the name", updateTime, 1, 2, 3, 4},
		"fifi":   {"Fifi", "Second of the name", updateTime, 1, 2, 3, 4},
		"loulou": {"Loulou", "Third of the name", updateTime, 1, 2, 3, 4},
		"donald": {"Donald", "Donald THE Duck", updateTime, 1, 2, 3, 4},
	})

	c, engine := gin.CreateTestContext(httptest.NewRecorder())
	engine = SetupRouter(&manager, engine)

	c.Request = httptest.NewRequest("GET", "/parkings/P+R?ids[]=donald&ids[]=picsou&ids[]=fifi", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(http.StatusOK, w.Code)

	response := ParkingsResponse{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)

	parkings := response.Parkings
	sort.Sort(ByParkingResponseId(parkings))
	require.Len(parkings, 2)
	assert.Equal("Donald", parkings[0].ID)
	assert.Equal("Fifi", parkings[1].ID)

	require.Len(response.Errors, 1)
	assert.Contains(response.Errors[0], "picsou")
}

func TestEquipmentsAPI(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	equipmentURI, err := url.Parse(fmt.Sprintf("file://%s/NET_ACCESS.XML", fixtureDir))
	require.Nil(err)

	var manager DataManager

	c, engine := gin.CreateTestContext(httptest.NewRecorder())
	engine = SetupRouter(&manager, engine)

	err = RefreshEquipments(&manager, *equipmentURI, defaultTimeout)
	assert.Nil(err)

	c.Request = httptest.NewRequest("GET", "/equipments", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)

	var response EquipmentsResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	require.NotNil(response.Equipments)
	require.NotEmpty(response.Equipments)
	assert.Len(response.Equipments, 3)
	assert.Empty(response.Error)
}

func TestFreeFloatingsAPIWithDataFromFile(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	// Load freefloatings from a json file
	uri, err := url.Parse(fmt.Sprintf("file://%s/vehicles.json", fixtureDir))
	require.Nil(err)
	reader, err := getFileWithFS(*uri)
	require.Nil(err)

	jsonData, err := ioutil.ReadAll(reader)
	require.Nil(err)

	data := &Data{}
	err = json.Unmarshal([]byte(jsonData), data)
	require.Nil(err)

	freeFloatings, err := LoadFreeFloatingData(data)
	require.Nil(err)

	var manager DataManager
	manager.UpdateFreeFloating(freeFloatings)
	c, engine := gin.CreateTestContext(httptest.NewRecorder())
	engine = SetupRouter(&manager, engine)

	// Request without any parameter (coord is mandatory)
	c.Request = httptest.NewRequest("GET", "/free_floatings", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(503, w.Code)

	var response FreeFloatingsResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	assert.Len(response.FreeFloatings, 0)
	assert.Equal("Bad request: coord is mandatory",response.Error)

	// Request with coord in parameter
	response.Error = ""
	c.Request = httptest.NewRequest("GET", "/free_floatings?coord=2.37715%3B48.846781", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	require.NotNil(response.FreeFloatings)
	assert.Len(response.FreeFloatings, 3)
	assert.Empty(response.Error)

	// Request with coord, count in parameter
	c.Request = httptest.NewRequest("GET", "/free_floatings?coord=2.37715%3B48.846781&count=2", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	require.NotNil(response.FreeFloatings)
	assert.Len(response.FreeFloatings, 2)
	assert.Empty(response.Error)

	// Request with coord, type[] in parameter
	c.Request = httptest.NewRequest("GET", "/free_floatings?coord=2.37715%3B48.846781&type[]=BIKE&type[]=toto", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	require.NotNil(response.FreeFloatings)
	assert.Len(response.FreeFloatings, 1)

	// Verify attributes
	assert.Equal("cG9ueTpCSUtFOjEwMDQ0MQ==", response.FreeFloatings[0].Id)
	assert.Equal("NSCBH3", response.FreeFloatings[0].PublicId)
	assert.Equal("http://test1", response.FreeFloatings[0].Deeplink)
	assert.Equal("Pony", response.FreeFloatings[0].ProviderName)
	assert.Equal([]string{"ELECTRIC"}, response.FreeFloatings[0].Attributes)
	assert.Equal(55, response.FreeFloatings[0].Battery)
	assert.Equal("ASSIST", response.FreeFloatings[0].Propulsion)
	assert.Equal(48.847232, response.FreeFloatings[0].Coord.Lat)
	assert.Equal(2.377601, response.FreeFloatings[0].Coord.Lon)
	assert.Equal(60.0, response.FreeFloatings[0].Distance)
}

func TestParameterTypes(t *testing.T) {
	// valid types : {"BIKE", "SCOOTER", "MOTORSCOOTER", "STATION", "CAR", "OTHER"}
	// As toto is not a valid type it will not be added in types
	assert := assert.New(t)
	p := FreeFloatingRequestParameter{}
	types := make([]string, 0)
	types = append(types, "STATION")
	types = append(types, "toto")
	types = append(types, "MOTORSCOOTER")
	types = append(types, "OTHER")

	updateParameterTypes(&p, types)
	assert.Len(p.types, 3)
}

func TestVehicleOccupanciesAPIWithDataFromFile(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	var manager DataManager
	loc, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	// Load StopPoints from file .../mapping_stops.csv
	uri, err := url.Parse(fmt.Sprintf("file://%s/", fixtureDir))
	stopPoints, err := LoadStopPoints(*uri, defaultTimeout)
	require.Nil(err)
	manager.InitStopPoint(stopPoints)
	assert.Equal(len(*manager.stopPoints), 25)

	// Load courses from file .../extraction_courses.csv
	uri, err = url.Parse(fmt.Sprintf("file://%s/", fixtureDir))
	courses, err := LoadCourses(*uri, defaultTimeout)
	require.Nil(err)
	manager.InitCourse(courses)
	assert.Equal(len(*manager.courses), 1)
	coursesFor40 := (*manager.courses)["40"]
	assert.Equal(len(coursesFor40), 310)

	// Load RouteSchedules from file
	uri, err = url.Parse(fmt.Sprintf("file://%s/route_schedules.json", fixtureDir))
	require.Nil(err)
	reader, err := getFileWithFS(*uri)
	require.Nil(err)

	jsonData, err := ioutil.ReadAll(reader)
	require.Nil(err)

	navitiaRoutes := &NavitiaRoutes{}
	err = json.Unmarshal([]byte(jsonData), navitiaRoutes)
	require.Nil(err)
	sens := 0
	startIndex := 1
	routeSchedules := LoadRouteSchedulesData(startIndex, navitiaRoutes, sens, loc)
	manager.InitRouteSchedule(routeSchedules)
	assert.Equal(len(*manager.routeSchedules), 141)

	// Load prediction from a file
	uri, err = url.Parse(fmt.Sprintf("file://%s/predictions.json", fixtureDir))
	require.Nil(err)
	reader, err = getFileWithFS(*uri)
	require.Nil(err)

	jsonData, err = ioutil.ReadAll(reader)
	require.Nil(err)

	predicts := &PredictionData{}
	err = json.Unmarshal([]byte(jsonData), predicts)
	require.Nil(err)
	predictions := LoadPredictionsData(predicts, loc)
	assert.Equal(len(predictions), 65)

	occupanciesWithCharge := CreateOccupanciesFromPredictions(&manager, predictions)
	manager.UpdateVehicleOccupancies(occupanciesWithCharge)
	assert.Equal(len(*manager.vehicleOccupancies),35)

	c, engine := gin.CreateTestContext(httptest.NewRecorder())
	engine = SetupRouter(&manager, engine)

	// Request without any parameter (Date with default value = Now().Format("20060102"))
	response := VehicleOccupanciesResponse{}
	c.Request = httptest.NewRequest("GET", "/vehicle_occupancies", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	require.Nil(response.VehicleOccupancies)
	assert.Len(response.VehicleOccupancies, 0)
	assert.Empty(response.Error)

	c.Request = httptest.NewRequest("GET", "/vehicle_occupancies?date=20210118", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	require.NotNil(response.VehicleOccupancies)
	assert.Len(response.VehicleOccupancies, 35)
	assert.Empty(response.Error)

	resp := VehicleOccupanciesResponse{}
	c.Request = httptest.NewRequest("GET", "/vehicle_occupancies?date=20210118&vehiclejourney_id=vehicle_journey:0:123713792-1", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.Nil(err)
	require.NotNil(resp.VehicleOccupancies)
	assert.Len(resp.VehicleOccupancies, 7)
	assert.Empty(resp.Error)

	resp = VehicleOccupanciesResponse{}
	c.Request = httptest.NewRequest("GET",
	"/vehicle_occupancies?date=20210118&vehiclejourney_id=vehicle_journey:0:123713792-1&stop_id=stop_point:0:SP:80:4121", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.Nil(err)
	require.NotNil(resp.VehicleOccupancies)
	assert.Len(resp.VehicleOccupancies, 1)
	assert.Empty(resp.Error)

	require.Nil(err)
	assert.Equal(resp.VehicleOccupancies[0].VehicleJourneyId, "vehicle_journey:0:123713792-1")
	assert.Equal(resp.VehicleOccupancies[0].StopId, "stop_point:0:SP:80:4121")
	assert.Equal(resp.VehicleOccupancies[0].Direction, 0)
	assert.Equal(resp.VehicleOccupancies[0].DateTime.Format("20060102T150405"), "20210118T072200")
	assert.Equal(resp.VehicleOccupancies[0].Occupancy, 11)
}
