package main

import (
	"embed"
	"html/template"
	"io/fs"
	"net"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
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

type appConfig struct {
  AppUrl string
  AppPort int
}

var ac appConfig

func init() {
  logger, _ = zap.NewDevelopment()
  defer logger.Sync() // flushes buffer, if any

  // app configuration
  viper.SetDefault("APP_URL", "http://localhost")
  viper.SetDefault("APP_PORT", 3001)
  viper.SetEnvPrefix("WG")
  viper.AutomaticEnv()

  ac.AppUrl = viper.GetString("APP_URL")
  ac.AppPort = viper.GetInt("APP_PORT")
  logger.Sugar().Debugf("App will be available under: %s:%d", ac.AppUrl, ac.AppPort)
}

func main() {
  logger.Info("Starting up.")

  if os.Geteuid() > 0 {
    logger.Fatal("Must be run as root, exiting.")
  }

  r := gin.Default()
  r.Use(cors.Default())
  // Returning the filesystem containing all the html files
  sub, err := fs.Sub(spaFiles, "spa")
  if err != nil {
    logger.Fatal("Error during initial load of html templates")
  }
  t, err := loadTemplates(sub)
  if err != nil {
    logger.Sugar().Fatal(err)
  }
  r.SetHTMLTemplate(t)

  r.GET("/", indexHandler)
  r.GET("/dashboard", dashboardHandler)
  r.GET("/info", wireguardHandler)
  r.Run(":3001")
}

func loadTemplates(f fs.FS) (*template.Template, error) {
  logger.Debug("Loading Templates")
  t := template.New("")
  fs.WalkDir(f, ".", func(path string, d fs.DirEntry, err error) error {
    if err != nil {
      logger.Sugar().Fatal(err)
    }
    // Check if its an html file, if nope skip
    if !strings.HasSuffix(path, ".html") {
			return nil
		}
    // Read that file
    h, err := fs.ReadFile(f, path)
		if err != nil {
      logger.Sugar().Fatal(err)
		}
    // Generate a new template from it
		t, err = t.New(path).Parse(string(h))
		if err != nil {
      logger.Sugar().Fatal(err)
		}
    return nil
  })

  return t, nil
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
  c.HTML(200, "index.html", ac)
}
