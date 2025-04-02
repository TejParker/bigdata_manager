<template>
  <div class="page-container">
    <a-card>
      <template #title>
        <a-space>
          <a-typography-title :heading="5" style="margin: 0">服务列表</a-typography-title>
        </a-space>
      </template>
      <template #extra>
        <a-space>
          <a-button type="primary" @click="openAddServiceModal">
            <template #icon>
              <icon-plus />
            </template>
            添加服务
          </a-button>
          <a-button @click="fetchServices">
            <template #icon>
              <icon-refresh />
            </template>
            刷新
          </a-button>
        </a-space>
      </template>

      <!-- 过滤条件 -->
      <a-form :model="filterForm" layout="inline" class="mb-4">
        <a-form-item field="cluster_id" label="集群">
          <a-select
            v-model="filterForm.cluster_id"
            placeholder="选择集群"
            allow-clear
            style="width: 160px"
          >
            <a-option
              v-for="cluster in clusters"
              :key="cluster.id"
              :value="cluster.id"
              :label="cluster.name"
            />
          </a-select>
        </a-form-item>
        <a-form-item field="service_type" label="服务类型">
          <a-select
            v-model="filterForm.service_type"
            placeholder="选择服务类型"
            allow-clear
            style="width: 160px"
          >
            <a-option value="HDFS">HDFS</a-option>
            <a-option value="YARN">YARN</a-option>
            <a-option value="HIVE">Hive</a-option>
            <a-option value="SPARK">Spark</a-option>
            <a-option value="KAFKA">Kafka</a-option>
            <a-option value="FLINK">Flink</a-option>
            <a-option value="ZOOKEEPER">ZooKeeper</a-option>
          </a-select>
        </a-form-item>
        <a-form-item field="status" label="状态">
          <a-select
            v-model="filterForm.status"
            placeholder="选择状态"
            allow-clear
            style="width: 160px"
          >
            <a-option value="RUNNING">运行中</a-option>
            <a-option value="STOPPED">已停止</a-option>
            <a-option value="INSTALLING">安装中</a-option>
            <a-option value="FAILED">失败</a-option>
          </a-select>
        </a-form-item>
        <a-form-item>
          <a-button type="primary" @click="fetchServices(1)">
            查询
          </a-button>
          <a-button style="margin-left: 8px" @click="resetFilter">
            重置
          </a-button>
        </a-form-item>
      </a-form>

      <!-- 服务列表 -->
      <a-table
        :data="services"
        :loading="loading"
        :pagination="pagination"
        @page-change="onPageChange"
        @page-size-change="onPageSizeChange"
        :bordered="false"
        stripe
      >
        <template #columns>
          <a-table-column title="服务名称" data-index="service_name">
            <template #cell="{ record }">
              <a-link @click="gotoServiceDetail(record.id)">
                {{ record.service_name }}
              </a-link>
            </template>
          </a-table-column>
          <a-table-column title="服务类型" data-index="service_type" />
          <a-table-column title="所属集群" data-index="cluster_name" />
          <a-table-column title="版本" data-index="version" />
          <a-table-column title="状态" data-index="status" width="120">
            <template #cell="{ record }">
              <a-tag
                :color="getStatusColor(record.status)"
              >
                {{ getStatusText(record.status) }}
              </a-tag>
            </template>
          </a-table-column>
          <a-table-column title="创建时间" data-index="created_at" />
          <a-table-column title="操作" width="200">
            <template #cell="{ record }">
              <a-space>
                <a-button
                  type="text"
                  size="small"
                  :disabled="record.status !== 'STOPPED'"
                  @click="startService(record.id)"
                >
                  启动
                </a-button>
                <a-button
                  type="text"
                  size="small"
                  :disabled="record.status !== 'RUNNING'"
                  @click="stopService(record.id)"
                >
                  停止
                </a-button>
                <a-popconfirm
                  position="left"
                  content="确定要删除该服务吗？"
                  @ok="deleteService(record.id)"
                >
                  <a-button
                    type="text"
                    status="danger"
                    size="small"
                    :disabled="record.status === 'RUNNING'"
                  >
                    删除
                  </a-button>
                </a-popconfirm>
              </a-space>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </a-card>

    <!-- 添加服务对话框 -->
    <a-modal
      v-model:visible="addServiceVisible"
      title="添加服务"
      @ok="submitAddService"
      @cancel="addServiceVisible = false"
      :ok-loading="submitting"
      unmount-on-close
    >
      <a-form
        ref="addServiceFormRef"
        :model="addServiceForm"
        :rules="addServiceRules"
        label-align="right"
        :label-col-props="{ span: 6 }"
        :wrapper-col-props="{ span: 18 }"
      >
        <a-form-item field="cluster_id" label="集群">
          <a-select
            v-model="addServiceForm.cluster_id"
            placeholder="选择集群"
            allow-clear
          >
            <a-option
              v-for="cluster in clusters"
              :key="cluster.id"
              :value="cluster.id"
              :label="cluster.name"
            />
          </a-select>
        </a-form-item>
        <a-form-item field="service_type" label="服务类型">
          <a-select
            v-model="addServiceForm.service_type"
            placeholder="选择服务类型"
            allow-clear
          >
            <a-option value="HDFS">HDFS</a-option>
            <a-option value="YARN">YARN</a-option>
            <a-option value="HIVE">Hive</a-option>
            <a-option value="SPARK">Spark</a-option>
            <a-option value="KAFKA">Kafka</a-option>
            <a-option value="FLINK">Flink</a-option>
            <a-option value="ZOOKEEPER">ZooKeeper</a-option>
          </a-select>
        </a-form-item>
        <a-form-item field="service_name" label="服务名称">
          <a-input
            v-model="addServiceForm.service_name"
            placeholder="输入服务名称"
            allow-clear
          />
        </a-form-item>
        <a-form-item field="version" label="版本">
          <a-input
            v-model="addServiceForm.version"
            placeholder="输入版本号，如：3.2.1"
            allow-clear
          />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { Message, Modal } from '@arco-design/web-vue';
import request from '@/utils/request';

const router = useRouter();
const loading = ref(false);
const services = ref([]);
const clusters = ref([]);
const addServiceFormRef = ref(null);

// 分页
const pagination = reactive({
  total: 0,
  current: 1,
  pageSize: 10,
  showTotal: true,
  showPageSize: true,
});

// 过滤表单
const filterForm = reactive({
  cluster_id: '',
  service_type: '',
  status: '',
});

// 添加服务表单
const addServiceVisible = ref(false);
const submitting = ref(false);
const addServiceForm = reactive({
  cluster_id: '',
  service_type: '',
  service_name: '',
  version: '',
});

// 添加服务表单验证规则
const addServiceRules = {
  cluster_id: [
    { required: true, message: '请选择集群' },
  ],
  service_type: [
    { required: true, message: '请选择服务类型' },
  ],
  service_name: [
    { required: true, message: '请输入服务名称' },
    { maxLength: 50, message: '服务名称最多50个字符' },
  ],
  version: [
    { required: true, message: '请输入版本' },
    { maxLength: 20, message: '版本最多20个字符' },
  ],
};

// 获取集群列表
const fetchClusters = async () => {
  try {
    const response = await request.get('/clusters', {
      params: {
        page_size: 100,
      },
    });
    if (response.data.success) {
      clusters.value = response.data.data.items;
    }
  } catch (error) {
    console.error('获取集群列表失败', error);
  }
};

// 获取服务列表
const fetchServices = async (page = pagination.current) => {
  loading.value = true;
  try {
    const params = {
      page,
      page_size: pagination.pageSize,
      ...filterForm,
    };
    
    const response = await request.get('/services', { params });
    
    if (response.data.success) {
      services.value = response.data.data.items;
      pagination.total = response.data.data.total;
      pagination.current = page;
    }
  } catch (error) {
    console.error('获取服务列表失败', error);
    Message.error('获取服务列表失败');
  } finally {
    loading.value = false;
  }
};

// 重置过滤条件
const resetFilter = () => {
  filterForm.cluster_id = '';
  filterForm.service_type = '';
  filterForm.status = '';
  fetchServices(1);
};

// 分页变化
const onPageChange = (page) => {
  fetchServices(page);
};

// 每页条数变化
const onPageSizeChange = (pageSize) => {
  pagination.pageSize = pageSize;
  fetchServices(1);
};

// 根据状态获取颜色
const getStatusColor = (status) => {
  switch (status) {
    case 'RUNNING':
      return 'green';
    case 'STOPPED':
      return 'gray';
    case 'INSTALLING':
      return 'blue';
    case 'FAILED':
      return 'red';
    default:
      return 'gray';
  }
};

// 状态文本映射
const getStatusText = (status) => {
  switch (status) {
    case 'RUNNING':
      return '运行中';
    case 'STOPPED':
      return '已停止';
    case 'INSTALLING':
      return '安装中';
    case 'FAILED':
      return '失败';
    default:
      return status;
  }
};

// 跳转到服务详情
const gotoServiceDetail = (serviceId) => {
  router.push(`/services/${serviceId}`);
};

// 打开添加服务对话框
const openAddServiceModal = () => {
  Object.keys(addServiceForm).forEach(key => {
    addServiceForm[key] = '';
  });
  addServiceVisible.value = true;
};

// 提交添加服务
const submitAddService = () => {
  addServiceFormRef.value.validate(async (valid) => {
    if (valid) {
      submitting.value = true;
      
      try {
        const response = await request.post('/services', addServiceForm);
        
        if (response.data.success) {
          Message.success('添加服务成功');
          addServiceVisible.value = false;
          fetchServices();
        } else {
          Message.error(response.data.message || '添加服务失败');
        }
      } catch (error) {
        console.error('添加服务失败', error);
        Message.error('添加服务失败');
      } finally {
        submitting.value = false;
      }
    }
  });
};

// 启动服务
const startService = async (serviceId) => {
  try {
    const response = await request.post(`/services/${serviceId}/start`);
    
    if (response.data.success) {
      Message.success('服务启动命令已发送');
      fetchServices();
    } else {
      Message.error(response.data.message || '启动服务失败');
    }
  } catch (error) {
    console.error('启动服务失败', error);
    Message.error('启动服务失败');
  }
};

// 停止服务
const stopService = async (serviceId) => {
  try {
    const response = await request.post(`/services/${serviceId}/stop`);
    
    if (response.data.success) {
      Message.success('服务停止命令已发送');
      fetchServices();
    } else {
      Message.error(response.data.message || '停止服务失败');
    }
  } catch (error) {
    console.error('停止服务失败', error);
    Message.error('停止服务失败');
  }
};

// 删除服务
const deleteService = async (serviceId) => {
  try {
    const response = await request.delete(`/services/${serviceId}`);
    
    if (response.data.success) {
      Message.success('服务删除成功');
      fetchServices();
    } else {
      Message.error(response.data.message || '删除服务失败');
    }
  } catch (error) {
    console.error('删除服务失败', error);
    Message.error('删除服务失败');
  }
};

// 初始化
onMounted(async () => {
  await fetchClusters();
  await fetchServices();
});
</script>

<style scoped>
.page-container {
  padding: 16px;
}
</style> 