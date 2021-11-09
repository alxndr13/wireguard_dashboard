package main

import (
	"embed"
	_ "embed"
	"io/fs"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.zx2c4.com/wireguard/wgctrl"
)

var logger *zap.Logger
//go:embed spa
var spaFiles embed.FS


func init() {
  logger, _ = zap.NewProduction()
  defer logger.Sync() // flushes buffer, if any
}

func main() {
  logger.Info("Starting up.")

  if os.Geteuid() > 0 {
    logger.Fatal("Must be run as root, exiting.")
  }
  r := gin.Default()
  r.Use(cors.Default())
  r.GET("/", indexHandler)
  sub, err := fs.Sub(spaFiles, "spa")
  if err != nil {
    logger.Fatal("Error during loading of html templates")
  }
  r.StaticFS("/dashboard", http.FS(sub))
  r.GET("/info", wireguardHandler)
  r.Run(":3001")
}

func indexHandler(c *gin.Context) {
  c.JSON(200, gin.H{"message": "wireguard status api"})
}

func wireguardHandler(c *gin.Context) {


  wgc, err := wgctrl.New()
  if err != nil {
    c.JSON(500, gin.H{"error": "could not create wireguard client"})
  }
  devices, err := wgc.Devices()

  type peerInfo struct {
    LastHandshake string
    Endpoint *net.UDPAddr
  }
  type deviceInfo struct {
    DeviceName string
    PeerInfo []peerInfo
  }

  di := []deviceInfo{}

  for _,v := range devices {
    tdi := deviceInfo{}
    tdi.DeviceName = v.Name
    for _,j := range v.Peers {
      tPi := peerInfo{}
      // Format Time to a readable string which can be used in the frontend.
      tPi.LastHandshake = j.LastHandshakeTime.Format(time.RFC1123)
      tPi.Endpoint = j.Endpoint
      tdi.PeerInfo = append(tdi.PeerInfo, tPi)
    }
    di = append(di, tdi)
  }
  c.JSON(200, di)
}
