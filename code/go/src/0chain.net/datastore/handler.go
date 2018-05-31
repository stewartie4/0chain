package datastore

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"0chain.net/common"
)

/*EntityProvider - returns an entity */
type EntityProvider func() Entity

/*JSONEntityReqResponderF - a handler that takes a JSON request and responds with a json response
* Useful for GET operation where the input is coming via url parameters
 */
type JSONEntityReqResponderF func(ctx context.Context, entity Entity) (interface{}, error)

/*ToJSONEntityReqResponse - Similar to ToJSONReqResponse except it takes an EntityProvider
* that returns an interface into which the incoming request json is unmarshalled
* Avoids extra map creation and also wiring it manually from the map to the entity object
 */
func ToJSONEntityReqResponse(handler JSONEntityReqResponderF, entityMetadata EntityMetadata) common.ReqRespHandlerf {
	return func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-type")
		if !strings.HasPrefix(contentType, "application/json") {
			http.Error(w, "Header Content-type=application/json not found", 400)
			return
		}
		decoder := json.NewDecoder(r.Body)
		entity := entityMetadata.Instance()
		err := decoder.Decode(entity)
		if err != nil {
			http.Error(w, "Error decoding json", 500)
			return
		}
		ctx := r.Context()
		data, err := handler(ctx, entity)
		common.Respond(w, data, err)
	}
}

/*PrintEntityHandler - handler that prints the received entity */
func PrintEntityHandler(ctx context.Context, entity Entity) (interface{}, error) {
	fmt.Printf("%v: %v\n", entity.GetEntityName(), ToJSON(entity))
	return nil, nil
}
