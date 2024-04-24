package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/sts"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

const MaxFileSize = 1024 * 1024 * 100 // 100MB
const UploadPath = "./uploads"        // 上传文件存储路径

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func main() {
	router := setupRouter()
	log.Fatal(router.Run(":8090"))
}

func setupRouter() *gin.Engine {
	router := gin.Default()
	router.Use(CORSMiddleware()) // 添加这行代码
	router.POST("/upload", handleUpload)
	router.POST("/upload/chunk", handleChunkUpload)
	router.POST("/upload/check/", handleCheckChunk) // 添加这行代码
	router.POST("/upload/merge", handleCompleteUpload)
	router.GET("/oss/stsToken", handleStsToken) // 实际产品中。这里要求登录才能调用。

	return router
}

func handleUpload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查文件大小
	if file.Size > MaxFileSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File size exceeds the limit"})
		return
	}

	// 创建上传目录
	err = os.MkdirAll(UploadPath, os.ModePerm)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory"})
		return
	}

	// 生成文件名
	filename := filepath.Join(UploadPath, file.Filename)

	// 创建目标文件
	out, err := os.Create(filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create the file"})
		return
	}
	defer out.Close()

	// 复制文件内容
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open the uploaded file"})
		return
	}
	defer src.Close()

	_, err = io.Copy(out, src)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save the uploaded file"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "File uploaded successfully"})
}

func handleChunkUpload(c *gin.Context) {
	err := c.Request.ParseMultipartForm(MaxFileSize)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	chunkNumber, err := strconv.Atoi(c.Request.FormValue("chunkNumber"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// chunkTotal, err := strconv.Atoi(c.Request.FormValue("chunkTotal"))
	// if err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	// 	return
	// }
	filename := c.Request.FormValue("filename")
	hasher := sha256.New()
	hasher.Write([]byte(filename))
	println(filename)
	hashedFilename := hex.EncodeToString(hasher.Sum(nil))

	tempDir := filepath.Join(UploadPath, "temp", hashedFilename)
	println(tempDir)

	err = os.MkdirAll(tempDir, os.ModePerm)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create temporary directory"})
		return
	}

	chunkFilename := filepath.Join(tempDir, fmt.Sprintf("%d", chunkNumber))
	out, err := os.Create(chunkFilename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create temporary chunk file"})
		return
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save the uploaded chunk"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Chunk uploaded successfully"})
}

func handleCompleteUpload(c *gin.Context) {
	filename := c.Request.FormValue("filename")
	println(filename)
	hasher := sha256.New()
	hasher.Write([]byte(filename))
	hashedFilename := hex.EncodeToString(hasher.Sum(nil))
	tempDir := filepath.Join(UploadPath, "temp", hashedFilename)
	chunkFiles, err := ioutil.ReadDir(tempDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read temporary directory:" + tempDir})
		return
	}

	destinationPath := filepath.Join(UploadPath, filename)
	destinationFile, err := os.Create(destinationPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create destination file"})
		return
	}
	defer destinationFile.Close()

	for _, chunkFile := range chunkFiles {
		chunkPath := filepath.Join(tempDir, chunkFile.Name())
		chunkData, err := ioutil.ReadFile(chunkPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read temporary chunk file"})
			return
		}

		_, err = destinationFile.Write(chunkData)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write to destination file"})
			return
		}

		err = os.Remove(chunkPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove temporary chunk file"})
			return
		}
	}

	err = os.Remove(tempDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove temporary directory"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "File upload completed successfully"})
}

func handleCheckChunk(c *gin.Context) {
	filename := c.Request.FormValue("filename")
	chunkNumber := c.Request.FormValue("chunkNumber")

	hasher := sha256.New()
	hasher.Write([]byte(filename))
	hashedFilename := hex.EncodeToString(hasher.Sum(nil))
	tempDir := filepath.Join(UploadPath, "temp", hashedFilename)

	chunkFilePath := filepath.Join(tempDir, chunkNumber)
	if _, err := os.Stat(chunkFilePath); os.IsNotExist(err) {
		c.JSON(http.StatusOK, gin.H{"exist": false})
	} else {
		c.JSON(http.StatusOK, gin.H{"exist": true})
	}
}

var (
	accessKeyID,
	accessKeySecret,
	roleArn string
)

func init() {
	// 加载.env文件
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Failed to load .env file")
	}

	// 从环境变量中读取阿里云访问密钥
	accessKeyID = os.Getenv("ACCESS_KEY_ID")
	accessKeySecret = os.Getenv("ACCESS_KEY_SECRET")
	roleArn = os.Getenv("ROLE_ARN")
}

func handleStsToken(c *gin.Context) {

	// 创建STS客户端
	client, err := sts.NewClientWithAccessKey("cn-beijing", accessKeyID, accessKeySecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create STS client"})
		return
	}

	// 构建AssumeRole请求
	request := sts.CreateAssumeRoleRequest()
	request.Scheme = "https"
	request.RoleArn = roleArn
	request.RoleSessionName = "oss-upload-in-frontend"

	// 发起AssumeRole请求
	response, err := client.AssumeRole(request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assume role"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"AccessKeyId": response.Credentials.AccessKeyId, "AccessKeySecret": response.Credentials.AccessKeySecret, "SecurityToken": response.Credentials.SecurityToken, "Expiration": response.Credentials.Expiration})

}
