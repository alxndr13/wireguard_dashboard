package main

import (
	"embed"
	"io/fs"
	"net"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.zx2c4.com/wireguard/wgctrl"
)

var logger *zap.Logger

// Embedding all the files in the `spa` folder
//go:embed spa
var spaFiles embed.FS

type peerInfo struct {
  LastHandshake string
  Endpoint *net.UDPAddr
}

type deviceInfo struct {
  DeviceName string
  PeerInfo []peerInfo
}

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

  // Returning the filesystem containing all the html files
  sub, err := fs.Sub(spaFiles, "spa")
  if err != nil {
    logger.Fatal("Error during loading of html templates")
  }
  // delivering the fs under /dashboard
  r.GET("/dashboard", dashboardHandler)
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
  if err != nil {
    c.JSON(500, gin.H{"error": "could not get wireguard devices"})
  }

  // Holds the device info of this client
  di := []deviceInfo{}

  for _,v := range devices {
    // Temporary device info buffer to hold the current device info
    tdi := deviceInfo{}
    tdi.DeviceName = v.Name
    for _,j := range v.Peers {
      // Temporary peer info
      tpi := peerInfo{}
      // Format Time to a readable string which can be used in the frontend.
      tpi.LastHandshake = j.LastHandshakeTime.Format(time.RFC1123)
      tpi.Endpoint = j.Endpoint
      tdi.PeerInfo = append(tdi.PeerInfo, tpi)
    }
    // append to the clients list of devices
    di = append(di, tdi)
  }
  c.JSON(200, di)
}

func dashboardHandler(c *gin.Context) {
  c.HTML(200, "index.html")
}
