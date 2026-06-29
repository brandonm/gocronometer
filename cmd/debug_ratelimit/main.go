package main
import ("context";"fmt";"os";"time";"github.com/jrmycanady/gocronometer")
func main(){
  c:=gocronometer.NewClient(nil)
  ctx,cancel:=context.WithTimeout(context.Background(),60*time.Second); defer cancel()
  if err:=c.Login(ctx,os.Getenv("CRONOMETER_EMAIL"),os.Getenv("CRONOMETER_PASSWORD"));err!=nil{fmt.Println(err);return}
  tc,_:=c.GetThrottleConfig(ctx)
  if tc!=nil{ if a,ok:=tc.APIs["gwt_rpc"];ok{ fmt.Printf("published gwt_rpc: %.0f rps -> limiter %.0f rps (interval ~%.0fms)\n", a.Global.RPS, a.Global.RPS*gocronometer.ThrottleSafetyFactor, 1000/(a.Global.RPS*gocronometer.ThrottleSafetyFactor)) } }
  t0:=time.Now()
  for i:=0;i<6;i++{ c.GetFood(ctx, 21456763) }
  fmt.Printf("6 sequential GWT calls took %v\n", time.Since(t0))
}
