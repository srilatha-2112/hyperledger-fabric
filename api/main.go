package main

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"encoding/asn1"
	"encoding/json"
	"encoding/pem"
	"errors"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type Account struct {
	DEALERID    string `json:"DEALERID"`
	MSISDN      string `json:"MSISDN"`
	MPIN        string `json:"MPIN"`
	BALANCE     int64  `json:"BALANCE"`
	STATUS      string `json:"STATUS"`
	TRANSAMOUNT int64  `json:"TRANSAMOUNT"`
	TRANSTYPE   string `json:"TRANSTYPE"`
	REMARKS     string `json:"REMARKS"`
}

type History struct {
	TxID      string   `json:"txId"`
	Value     *Account `json:"value,omitempty"`
	IsDelete  bool     `json:"isDelete"`
	Timestamp int64    `json:"timestamp"`
}

var gw *client.Gateway
var contract *client.Contract

func mustEnv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("missing %s", k)
	}
	return v
}

func readFile(p string) []byte {
	b, err := os.ReadFile(p)
	if err != nil {
		log.Fatalf("read %s: %v", p, err)
	}
	return b
}

func pemBlock(b []byte) *pem.Block {
	p, _ := pem.Decode(b)
	if p == nil {
		log.Fatal("pem decode failed")
	}
	return p
}

func privateKeyFromPEM(keyPEM []byte) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return nil, errors.New("pem decode failed")
	}
	if k, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if ec, ok := k.(*ecdsa.PrivateKey); ok {
			return ec, nil
		}
		return nil, errors.New("not ecdsa")
	}
	return x509.ParseECPrivateKey(block.Bytes)
}

func connect() {
	peerEndpoint := mustEnv("PEER_ENDPOINT")
	gatewayPeer := mustEnv("GATEWAY_PEER")
	mspID := mustEnv("MSP_ID")
	channel := mustEnv("CHANNEL_NAME")
	ccName := mustEnv("CHAINCODE_NAME")
	tlsCertPath := mustEnv("TLS_CERT_PATH")
	certPath := mustEnv("CERT_PATH")
	keyPath := mustEnv("KEY_PATH")

	creds, err := credentials.NewClientTLSFromFile(tlsCertPath, gatewayPeer)
	if err != nil {
		log.Fatal(err)
	}
	conn, err := grpc.Dial(peerEndpoint, grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatal(err)
	}

	cert, err := identity.CertificateFromPEM(readFile(certPath))
	if err != nil {
		log.Fatal(err)
	}
	id, err := identity.NewX509Identity(mspID, cert)
	if err != nil {
		log.Fatal(err)
	}
	priv, err := privateKeyFromPEM(readFile(keyPath))
	if err != nil {
		log.Fatal(err)
	}
	type ecdsaSig struct{ R, S *big.Int }
	sign := func(digest []byte) ([]byte, error) {
		r, s, err := ecdsa.Sign(rand.Reader, priv, digest)
		if err != nil {
			return nil, err
		}
		return asn1.Marshal(ecdsaSig{r, s})
	}

	gw, err = client.Connect(id, client.WithSign(sign), client.WithClientConnection(conn), client.WithEvaluateTimeout(10*time.Second), client.WithEndorseTimeout(10*time.Second), client.WithSubmitTimeout(10*time.Second), client.WithCommitStatusTimeout(10*time.Second))
	if err != nil {
		log.Fatal(err)
	}
	network := gw.GetNetwork(channel)
	contract = network.GetContract(ccName)
}

func main() {
	connect()
	defer gw.Close()

	r := gin.Default()
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

	r.GET("/assets", func(c *gin.Context) {
		res, err := contract.EvaluateTransaction("GetAllAssets")
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		var out []Account
		if len(res) > 0 {
			if err := json.Unmarshal(res, &out); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
		}
		c.JSON(200, out)
	})

	r.GET("/assets/:msisdn", func(c *gin.Context) {
		msisdn := c.Param("msisdn")
		res, err := contract.EvaluateTransaction("ReadAsset", msisdn)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		var a Account
		if err := json.Unmarshal(res, &a); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, a)
	})

	r.GET("/assets/:msisdn/history", func(c *gin.Context) {
		msisdn := c.Param("msisdn")
		res, err := contract.EvaluateTransaction("GetAssetHistory", msisdn)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		var h []History
		if len(res) > 0 {
			if err := json.Unmarshal(res, &h); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
		}
		c.JSON(200, h)
	})

	r.POST("/assets", func(c *gin.Context) {
		var a Account
		if err := c.BindJSON(&a); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		_, err := contract.SubmitTransaction("CreateAsset", a.DEALERID, a.MSISDN, a.MPIN, strconv.FormatInt(a.BALANCE, 10), a.STATUS, strconv.FormatInt(a.TRANSAMOUNT, 10), a.TRANSTYPE, a.REMARKS)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(201, gin.H{"message": "created", "msisdn": a.MSISDN})
	})

	r.PUT("/assets/:msisdn", func(c *gin.Context) {
		msisdn := c.Param("msisdn")
		var a Account
		if err := c.BindJSON(&a); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		if a.MSISDN == "" {
			a.MSISDN = msisdn
		}
		_, err := contract.SubmitTransaction("UpdateAsset", a.DEALERID, a.MSISDN, a.MPIN, strconv.FormatInt(a.BALANCE, 10), a.STATUS, strconv.FormatInt(a.TRANSAMOUNT, 10), a.TRANSTYPE, a.REMARKS)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"message": "updated", "msisdn": a.MSISDN})
	})

	r.DELETE("/assets/:msisdn", func(c *gin.Context) {
		msisdn := c.Param("msisdn")
		_, err := contract.SubmitTransaction("DeleteAsset", msisdn)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"message": "deleted", "msisdn": msisdn})
	})

	addr := os.Getenv("API_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
