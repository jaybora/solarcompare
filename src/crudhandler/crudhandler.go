package crudhandler
//
//import (
//	"net/http"
//	"appengine"
//)
//
//func CrudHandler(w http.ResponseWriter, r *http.Request, keyFunc func(string, bool) *string) {
//	key := keyFunc(r.URL.String(), appengine.IsDevAppServer())
//	
//	if key == "" {
//		http.Error(w, fmt.Sprintf("No plant specified"), 
//			http.StatusBadRequest)
//		return	
//	}
//	
//	// If pvdata is specified then go to that handler
//	if strings.Contains(r.URL.String(), "pvdata") {
//		pvdatahandler.Pvdatahandler(w, r)
//		return
//	}
//	
//    switch r.Method {
//        case "GET": handleGet(w, r, &plantkey)
//        case "POST": handlePut(w, r, &plantkey)
//        case "PUT": handlePut(w, r, &plantkey)
//        case "DELETE": handleDelete(w, r, &plantkey)
//    }
//}
//
//func handleGet(w http.ResponseWriter, r *http.Request, plantkey *string) {
//	c := appengine.NewContext(r)
//	g := goon.NewGoon(r)
//	ps := plantdata.Plant{PlantKey:*plantkey}	
//		
//	if err := g.Get(&ps); err != nil {
//		c.Infof("Could not get plant for %s from datastore due to %s", *plantkey, err.Error())
//		http.Error(w, fmt.Sprintf("Plant not found"), 
//			http.StatusNotFound)
//		return
//	}
//	json, err := ps.ToJson()
//	if err != nil {
//		c.Infof("Error in marshalling to json for plantkey %s, err %s", *plantkey, err.Error())
//	}
//	w.Write(json)
//}
//
//func handlePut(w http.ResponseWriter, r *http.Request, plantkey *string) {
//	c := appengine.NewContext(r)
//	g := goon.NewGoon(r)
//	jsonbytes, _ := ioutil.ReadAll(r.Body)
//	ps, err := plantdata.ToPlant(&jsonbytes)
//	if err != nil {
//		http.Error(w, fmt.Sprintf("Plant data could not be unmarshalled"), 
//			http.StatusBadRequest)
//		return		
//	}
//	
//	ps.PlantKey = *plantkey
//
//	if err := g.Put(&ps); err != nil {
//		c.Errorf("Could not write plant data for plant %s: %s, %s", *plantkey, ps, err.Error())
//		http.Error(w, fmt.Sprintf("Plant data could be stored"), 
//			http.StatusInternalServerError)
//		return
//	}
//	w.Write([]byte("Ok"))
//}
//
//func handleDelete(w http.ResponseWriter, r *http.Request, plantkey *string) {
//	c := appengine.NewContext(r)
//	g := goon.NewGoon(r)
//	
//	if err := g.Delete(g.Key(&plantdata.Plant{PlantKey:*plantkey})); err != nil {
//		c.Errorf("Could not write delete plant %s: %s", *plantkey, err.Error())
//		http.Error(w, fmt.Sprintf("Plant could not be deleted"), 
//			http.StatusInternalServerError)
//		return
//	}
//	w.Write([]byte("Ok"))
//}
//
//
