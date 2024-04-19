<template>
  <div>
    <h1>自己实现分片上传</h1>
    <hr />
    <h1>普通文件上传</h1>
    <input type="file" @change="handleFileChange" />
    <button @click="uploadFile">Upload</button>
    <span
      >cost<strong>{{ cost1 }}</strong
      >秒</span
    >
    <div>
      <h1>串行切片上传</h1>
      <input type="file" @change="handleFileChange" />
      <button @click="uploadFileByTrunk">Upload</button>
      <span
        >cost<strong>{{ cost2 }}</strong
        >秒</span
      >
    </div>
    <div>
      <h1>并行切片上传</h1>
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
    <h1>oss分片上传</h1>
    <hr />
  </div>
</template>
<style scoped>
strong {
  color: green;
}
</style>
<script>
import axios from "axios"; // 添加这行代码
const http = axios.create({
  baseURL: "http://localhost:8090", // 设置你的服务器地址
});
export default {
  data() {
    return {
      file: null,
      cost1: 0,
      cost2: 0,
      cost3: 0,
      cost4: 0,
    };
  },
  methods: {
    handleFileChange(event) {
      this.file = event.target.files[0];
    },
    uploadFile() {
      const formData = new FormData();
      formData.append("file", this.file);
      const start = Date.now();
      fetch("http://localhost:8090/upload", {
        method: "POST",
        body: formData,
      })
        .then((response) => response.json())
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
  },
};
</script>