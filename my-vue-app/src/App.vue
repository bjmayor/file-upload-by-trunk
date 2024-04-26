<template>
  <div id="container">
    <div id="left">
      <h1>自己实现分片上传</h1>
      <hr />
      <h1>大文件直传</h1>
      <input type="file" @change="handleFileChange" />
      <button @click="uploadFile">Upload</button>
      <span
        >cost<strong>{{ cost1 }}</strong
        >秒</span
      >
      <div>
        <h1>分片上传(串行)</h1>
        <input type="file" @change="handleFileChange" />
        <button @click="uploadFileByTrunk">Upload</button>
        <span
          >cost<strong>{{ cost2 }}</strong
          >秒</span
        >
      </div>
      <div>
        <h1>分片上传(并行)</h1>
        <input type="file" @change="handleFileChange" />
        <button @click="uploadFileByTrunkParel">Upload</button>
        <span
          >cost<strong>{{ cost3 }}</strong
          >秒</span
        >
      </div>
      <div>
        <h1>断点续传上传(模拟10%的概率上传失败)</h1>
        <input type="file" @change="handleFileChange" />
        <button @click="uploadFileByTrunkParelContiue">Upload</button>
        <span
          >cost<strong>{{ cost4 }}</strong
          >秒</span
        >
      </div>
    </div>
    <div id="right">
      <h1>oss实现分片上传</h1>
      <hr />

      <div>
        <h1>文件直传(其实就是分片上传-并行)</h1>
        <input type="file" @change="handleFileChange" />
        <button @click="uploadFileByOssDirect">Upload</button>
        <span
          >cost<strong>{{ cost5 }}</strong
          >秒</span
        >
      </div>
      <div>
        <h1>分片上传(串行)</h1>
        <input type="file" @change="handleFileChange" />
        <button @click="uploadFileByOssSerial">Upload</button>
        <span
          >cost<strong>{{ cost6 }}</strong
          >秒</span
        >
      </div>
      <div>
        <h1>分片上传(并行)</h1>
        <input type="file" @change="handleFileChange" />
        <button @click="uploadFileByOssParal">Upload</button>
        <span
          >cost<strong>{{ cost7 }}</strong
          >秒</span
        >
      </div>
    </div>
  </div>
</template>
<style scoped>
strong {
  color: green;
}
#container {
  display: flex;
}

#left {
  flex: 1;
}

#right {
  flex: 1;
}
</style>
<script>
import OSS from "ali-oss";
import axios from "axios"; // 添加这行代码
const http = axios.create({
  // baseURL: "http://localhost:8090", // directo to golang server
  baseURL: "http://upload.demo.com", // use nginx to proxy
});
export default {
  data() {
    return {
      file: null,
      cost1: 0, // 直传
      cost2: 0, // 串行切片上传
      cost3: 0, // 并行切片上传
      cost4: 0, // 断点续传上传
      cost5: 0, // oss 直传
      cost6: 0, // oss 串行切片上传
      cost7: 0, // oss 并行切片上传
      cost8: 0, // oss 断点续传上传
      accessKeyId: "",
      accessSerect: "",
      securityToken: "",
    };
  },
  mounted() {
    http.get("/oss/stsToken").then((response) => {
      const { AccessKeyId, AccessKeySecret, SecurityToken } = response.data;
      this.accessKeyId = AccessKeyId;
      this.accessSerect = AccessKeySecret;
      this.securityToken = SecurityToken;
    });
  },
  methods: {
    handleFileChange(event) {
      this.file = event.target.files[0];
    },
    uploadFile() {
      const formData = new FormData();
      formData.append("file", this.file);
      const start = Date.now();
      http
        .post("/upload", formData)
        .then((data) => {
          console.log(data);
          const end = Date.now();
          this.cost1 = (end - start) / 1000;
          // 处理上传成功的逻辑
        })
        .catch((error) => {
          console.error(error);
          // 处理上传失败的逻辑
        });
    },

    uploadFileByTrunk() {
      if (!this.file) {
        return;
      }
      const start = Date.now();
      const chunkSize = 2 * 1024 * 1024; // 每个分片的大小（这里设置为2MB）
      const totalChunks = Math.ceil(this.file.size / chunkSize); // 总分片数
      let currentChunk = 0; // 当前分片索引

      const uploadChunk = () => {
        const start = currentChunk * chunkSize;
        const end = Math.min(start + chunkSize, this.file.size);
        const chunk = this.file.slice(start, end);

        const formData = new FormData();
        formData.append("file", chunk);
        formData.append("filename", this.file.name);
        formData.append("chunkNumber", currentChunk);
        formData.append("totalChunks", totalChunks);

        http
          .post("/upload/chunk", formData)
          .then(() => {
            currentChunk++;
            if (currentChunk < totalChunks) {
              uploadChunk(); // 继续上传下一个分片
            } else {
              mergeChunks(); // 所有分片上传完成后，执行合并请求
            }
          })
          .catch((error) => {
            console.error("文件上传失败", error);
          });
      };

      const mergeChunks = () => {
        const formData = new FormData();
        formData.append("filename", this.file.name);

        http
          .post("/upload/merge", formData, {
            headers: {
              "Content-Type": "multipart/form-data",
            },
          })
          .then(() => {
            const end = Date.now();
            this.cost2 = (end - start) / 1000;
            console.log("文件上传完成");
          })
          .catch((error) => {
            console.error("文件合并失败", error);
          });
      };

      uploadChunk(); // 开始上传第一个分片
    },
    uploadFileByTrunkParel() {
      if (!this.file) {
        return;
      }
      const start = Date.now();
      const chunkSize = 2 * 1024 * 1024; // 每个分片的大小（这里设置为2MB）
      const totalChunks = Math.ceil(this.file.size / chunkSize); // 总分片数

      const uploadPromises = Array.from({ length: totalChunks }).map(
        (_, index) => {
          const start = index * chunkSize;
          const end = Math.min(start + chunkSize, this.file.size);
          const chunk = this.file.slice(start, end);

          const formData = new FormData();
          formData.append("file", chunk);
          formData.append("filename", this.file.name);
          formData.append("chunkNumber", index);
          formData.append("totalChunks", totalChunks);

          return http.post("/upload/chunk", formData);
        }
      );

      Promise.all(uploadPromises)
        .then(() => {
          console.log("所有分片上传完成");
          mergeChunks();
          // 这里可以添加合并文件的代码
        })
        .catch((error) => {
          console.error("分片上传失败", error);
        });

      const mergeChunks = () => {
        const formData = new FormData();
        formData.append("filename", this.file.name);

        http
          .post("/upload/merge", formData, {
            headers: {
              "Content-Type": "multipart/form-data",
            },
          })
          .then(() => {
            const end = Date.now();
            this.cost3 = (end - start) / 1000;
            console.log("文件上传完成");
          })
          .catch((error) => {
            console.error("文件合并失败", error);
          });
      };
    },

    async uploadFileByTrunkParelContiue() {
      if (!this.file) {
        return;
      }
      const start = Date.now();
      const chunkSize = 2 * 1024 * 1024; // 每个分片的大小（这里设置为2MB）
      const totalChunks = Math.ceil(this.file.size / chunkSize); // 总分片数

      for (let index = 0; index < totalChunks; index++) {
        const start = index * chunkSize;
        const end = Math.min(start + chunkSize, this.file.size);
        const chunk = this.file.slice(start, end);
        const formData = new FormData();
        formData.append("file", chunk);
        formData.append("filename", this.file.name);
        formData.append("chunkNumber", index);
        formData.append("totalChunks", totalChunks);
        const checkResponse = await http.post(
          `/upload/check/`,
          {
            filename: this.file.name,
            chunkNumber: index,
          },
          {
            headers: {
              "Content-Type": "multipart/form-data",
            },
          }
        );
        // 10% 的概率直接报错
        if (Math.random() < 0.1) {
          alert("上传失败, 请重试");
          return;
        }
        if (checkResponse.data.exist) {
          continue; // 说明这个块已经上传过，跳过这个块
        }

        await http.post("/upload/chunk", formData);
      }

      const mergeChunks = () => {
        const formData = new FormData();
        formData.append("filename", this.file.name);

        http
          .post("/upload/merge", formData, {
            headers: {
              "Content-Type": "multipart/form-data",
            },
          })
          .then(() => {
            const end = Date.now();
            this.cost4 = (end - start) / 1000;
            console.log("文件上传完成");
          })
          .catch((error) => {
            console.error("文件合并失败", error);
          });
      };

      console.log("所有分片上传完成");

      await mergeChunks();
      // 这里可以添加合并文件的代码
    },

    async uploadFileByOssDirect() {
      if (!this.file) {
        alert("请选择文件");
        return;
      }
      const start = Date.now();
      const client = new OSS({
        accessKeyId: this.accessKeyId,
        accessKeySecret: this.accessSerect,
        stsToken: this.securityToken,
        bucket: "bjmayor", // change to yours
        region: "oss-cn-beijing", // change to yours
      });

      const fileName = "demo/" + this.file.name;
      const result = await client.put(fileName, this.file); // 初始化分片上传
      console.log("文件上传成功", result);
      const end = Date.now();
      this.cost5 = (end - start) / 1000;
    },

    async uploadFileByOssSerial() {
      if (!this.file) {
        alert("请选择文件");
        return;
      }
      const start = Date.now();
      const client = new OSS({
        accessKeyId: this.accessKeyId,
        accessKeySecret: this.accessSerect,
        stsToken: this.securityToken,
        bucket: "bjmayor", // change to yours
        region: "oss-cn-beijing", // change to yours
      });

      const fileName = "demo/" + this.file.name;
      const fileSize = this.file.size;
      const chunkSize = 2 * 1024 * 1024; // 设置每个切片的大小，这里设置为2MB
      const chunks = Math.ceil(fileSize / chunkSize); // 计算切片总数
      const parts = [];
      const { uploadId } = await client.initMultipartUpload(fileName); // 初始化分片上传

      for (let i = 0; i < chunks; i++) {
        const start = i * chunkSize;
        const end = Math.min(fileSize, start + chunkSize);
        const partNumber = i + 1;

        // 读取分片内容

        const result = await client.uploadPart(
          fileName,
          uploadId,
          partNumber,
          this.file,
          start,
          end
        ); // 上传每个切片
        console.log(`第${partNumber}个切片上传成功`, result);
        // 将上传成功的分片信息添加到列表中
        parts.push({
          number: partNumber,
          etag: result.etag,
        });
      }

      const result = await client.completeMultipartUpload(
        fileName,
        uploadId,
        parts
      ); // 完成分片上传
      console.log("文件上传成功", result);
      const end = Date.now();
      this.cost6 = (end - start) / 1000;
    },
    async uploadFileByOssParal() {
      if (!this.file) {
        alert("请选择文件");
        return;
      }

      const start = Date.now();
      const client = new OSS({
        accessKeyId: this.accessKeyId,
        accessKeySecret: this.accessSerect,
        stsToken: this.securityToken,
        bucket: "bjmayor", // change to yours
        region: "oss-cn-beijing", // change to yours
      });

      const fileName = "demo/" + this.file.name;
      const result = await client.multipartUpload(fileName, this.file, {
        disabledMD5: false,
      }); // 初始化分片上传
      console.log("文件上传成功", result);
      const end = Date.now();
      this.cost7 = (end - start) / 1000;

      //   const start = Date.now();
      //   const client = new OSS({
      //     accessKeyId: this.accessKeyId,
      //     accessKeySecret: this.accessSerect,
      //     stsToken: this.securityToken,
      //     bucket: "bjmayor", // change to yours
      //     region: "oss-cn-beijing", // change to yours
      //   });

      //   const fileName = "demo/" + this.file.name;
      //   const fileSize = this.file.size;
      //   const chunkSize = 1 * 1024 * 1024; // 设置每个切片的大小，这里设置为1MB
      //   const totalChunks = Math.ceil(fileSize / chunkSize); // 计算切片总数
      //   const parts = [];
      //   const { uploadId } = await client.initMultipartUpload(fileName); // 初始化分片上传
      //   const uploadChunk = async (i) => {
      //     const start = i * chunkSize;
      //     const end = Math.min(fileSize, start + chunkSize);
      //     const partNumber = i + 1;

      //     // 读取分片内容
      //     const result = await client.uploadPart(
      //       fileName,
      //       uploadId,
      //       partNumber,
      //       this.file,
      //       start,
      //       end
      //     );

      //     parts[i] = {
      //       etag: result.etag,
      //       number: partNumber,
      //     };
      //   };

      //   for (let i = 0; i < totalChunks; ) {
      //     const chunkNumbers = Array.from(
      //       { length: Math.min(5, totalChunks - i) },
      //       (_, j) => i + j
      //     );
      //     i += chunkNumbers.length;
      //     await Promise.all(chunkNumbers.map(uploadChunk));
      //   }
      //   console.log("所有分片上传完成", parts);
      //   const result = await client.completeMultipartUpload(
      //     fileName,
      //     uploadId,
      //     parts
      //   ); // 完成分片上传
      //   console.log("文件上传成功", result);
      //   const end = Date.now();
      //   this.cost7 = (end - start) / 1000;
    },
  },
};
</script>