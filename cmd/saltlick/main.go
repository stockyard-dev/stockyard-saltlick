package main
import ("fmt";"log";"net/http";"os";"github.com/stockyard-dev/stockyard-saltlick/internal/server";"github.com/stockyard-dev/stockyard-saltlick/internal/store")
func main(){port:=os.Getenv("PORT");if port==""{port="9700"};dataDir:=os.Getenv("DATA_DIR");if dataDir==""{dataDir="./saltlick-data"}
db,err:=store.Open(dataDir);if err!=nil{log.Fatalf("saltlick: %v",err)};defer db.Close();srv:=server.New(db)
fmt.Printf("\n  Salt Lick — Self-hosted feature flag service\n  Dashboard:  http://localhost:%s/ui\n  API:        http://localhost:%s/api\n\n",port,port)
log.Printf("saltlick: listening on :%s",port);log.Fatal(http.ListenAndServe(":"+port,srv))}
