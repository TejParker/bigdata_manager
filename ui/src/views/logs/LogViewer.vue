<template>
  <div class="page-container">
    <a-row :gutter="16">
      <a-col :span="24">
        <a-card class="mb-4">
          <a-form :model="filterForm" layout="inline" class="mb-4">
            <!-- 过滤条件组 -->
            <a-form-item field="host_id" label="主机">
              <a-select
                v-model="filterForm.host_id"
                placeholder="选择主机"
                allow-clear
                style="width: 180px"
              >
                <a-option
                  v-for="host in hosts"
                  :key="host.id"
                  :value="host.id"
                  :label="host.hostname"
                />
              </a-select>
            </a-form-item>
            <a-form-item field="service_id" label="服务">
              <a-select
                v-model="filterForm.service_id"
                placeholder="选择服务"
                allow-clear
                style="width: 180px"
              >
                <a-option
                  v-for="service in services"
                  :key="service.id"
                  :value="service.id"
                  :label="service.service_name"
                />
              </a-select>
            </a-form-item>
            <a-form-item field="log_level" label="日志级别">
              <a-select
                v-model="filterForm.log_level"
                placeholder="选择级别"
                allow-clear
                style="width: 120px"
                multiple
              >
                <a-option
                  v-for="level in logLevels"
                  :key="level"
                  :value="level"
                  :label="level"
                />
              </a-select>
            </a-form-item>
            <a-form-item field="time_range" label="时间范围">
              <a-range-picker
                v-model="timeRange"
                show-time
                style="width: 380px"
              />
            </a-form-item>
            <a-form-item field="keyword" label="关键词">
              <a-input
                v-model="filterForm.keyword"
                placeholder="搜索关键词"
                allow-clear
                style="width: 180px"
              />
            </a-form-item>
            <a-form-item>
              <a-space>
                <a-button type="primary" @click="fetchLogs(1)">
                  查询
                </a-button>
                <a-button @click="resetFilter">
                  重置
                </a-button>
              </a-space>
            </a-form-item>
          </a-form>

          <!-- 日志统计图表区域 -->
          <div class="log-stats mb-4" v-if="showStats">
            <a-row :gutter="16">
              <a-col :span="8">
                <a-card class="mb-4 stat-card">
                  <template #title>日志级别分布</template>
                  <div ref="levelChartRef" style="height: 200px;"></div>
                </a-card>
              </a-col>
              <a-col :span="8">
                <a-card class="mb-4 stat-card">
                  <template #title>每日日志数量</template>
                  <div ref="dailyChartRef" style="height: 200px;"></div>
                </a-card>
              </a-col>
              <a-col :span="8">
                <a-card class="mb-4 stat-card">
                  <template #title>主机日志分布</template>
                  <div ref="hostChartRef" style="height: 200px;"></div>
                </a-card>
              </a-col>
            </a-row>
          </div>

          <!-- 工具栏 -->
          <div class="toolbar mb-4">
            <a-space>
              <a-button @click="showStats = !showStats">
                {{ showStats ? '隐藏统计信息' : '显示统计信息' }}
              </a-button>
              <a-button @click="exportLogs">
                <template #icon>
                  <icon-download />
                </template>
                导出日志
              </a-button>
              <a-button @click="fetchLogs">
                <template #icon>
                  <icon-refresh />
                </template>
                刷新
              </a-button>
            </a-space>
          </div>

          <!-- 日志列表 -->
          <a-table
            :data="logs"
            :loading="loading"
            :pagination="pagination"
            @page-change="onPageChange"
            @page-size-change="onPageSizeChange"
            :bordered="false"
            stripe
          >
            <template #columns>
              <a-table-column title="时间" data-index="timestamp" />
              <a-table-column title="级别" data-index="log_level" width="100">
                <template #cell="{ record }">
                  <a-tag
                    :color="getLevelColor(record.log_level)"
                    size="small"
                  >
                    {{ record.log_level }}
                  </a-tag>
                </template>
              </a-table-column>
              <a-table-column title="主机" data-index="hostname" />
              <a-table-column title="服务" data-index="service_name" />
              <a-table-column title="组件" data-index="component_type" />
              <a-table-column title="日志内容" data-index="message">
                <template #cell="{ record }">
                  <a-typography-paragraph
                    :ellipsis="{ rows: 2, showTooltip: true }"
                    style="margin-bottom: 0"
                  >
                    {{ record.message }}
                  </a-typography-paragraph>
                </template>
              </a-table-column>
              <a-table-column title="操作" width="80">
                <template #cell="{ record }">
                  <a-button
                    type="text"
                    size="small"
                    @click="viewLogDetail(record)"
                  >
                    详情
                  </a-button>
                </template>
              </a-table-column>
            </template>
          </a-table>
        </a-card>
      </a-col>
    </a-row>

    <!-- 日志详情对话框 -->
    <a-modal
      v-model:visible="logDetailVisible"
      title="日志详情"
      width="800px"
      @cancel="logDetailVisible = false"
    >
      <a-descriptions
        :data="logDetailDescriptions"
        layout="horizontal"
        :column="1"
        size="medium"
        bordered
        class="mb-4"
      />
      <a-typography-paragraph>
        <pre>{{ currentLog.message }}</pre>
      </a-typography-paragraph>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue';
import * as echarts from 'echarts';
import request from '@/utils/request';

// 数据状态
const logs = ref([]);
const hosts = ref([]);
const services = ref([]);
const logLevels = ref([]);
const loading = ref(false);
const showStats = ref(false);
const logStats = ref({
  level_stats: {},
  daily_stats: [],
  host_stats: []
});

// 图表引用
const levelChartRef = ref(null);
const dailyChartRef = ref(null);
const hostChartRef = ref(null);
let levelChart = null;
let dailyChart = null;
let hostChart = null;

// 日志详情状态
const logDetailVisible = ref(false);
const currentLog = ref({});

// 分页
const pagination = reactive({
  total: 0,
  current: 1,
  pageSize: 50,
  showTotal: true,
  showPageSize: true,
});

// 过滤表单
const filterForm = reactive({
  host_id: '',
  service_id: '',
  log_level: [],
  keyword: '',
});

// 时间范围，默认最近24小时
const defaultStartTime = new Date();
defaultStartTime.setHours(defaultStartTime.getHours() - 24);
const timeRange = ref([defaultStartTime, new Date()]);

// 日志详情描述
const logDetailDescriptions = computed(() => {
  if (!currentLog.value) return [];
  
  return [
    {
      label: '时间',
      value: currentLog.value.timestamp,
    },
    {
      label: '级别',
      value: currentLog.value.log_level,
      labelStyle: { width: '80px' },
    },
    {
      label: '主机',
      value: currentLog.value.hostname || '-',
    },
    {
      label: '服务',
      value: currentLog.value.service_name || '-',
    },
    {
      label: '组件',
      value: currentLog.value.component_type || '-',
    },
  ];
});

// 获取主机列表
const fetchHosts = async () => {
  try {
    const response = await request.get('/hosts', {
      params: { page_size: 100 }
    });
    if (response.data.success) {
      hosts.value = response.data.data.items;
    }
  } catch (error) {
    console.error('获取主机列表失败', error);
  }
};

// 获取服务列表
const fetchServices = async () => {
  try {
    const response = await request.get('/services', {
      params: { page_size: 100 }
    });
    if (response.data.success) {
      services.value = response.data.data.items;
    }
  } catch (error) {
    console.error('获取服务列表失败', error);
  }
};

// 获取日志级别列表
const fetchLogLevels = async () => {
  try {
    const response = await request.get('/log-levels');
    if (response.data.success) {
      logLevels.value = response.data.data;
    }
  } catch (error) {
    console.error('获取日志级别列表失败', error);
  }
};

// 获取日志统计信息
const fetchLogStats = async () => {
  try {
    const response = await request.get('/log-stats', {
      params: { days: 7 }
    });
    if (response.data.success) {
      logStats.value = response.data.data;
      // 初始化统计图表
      initCharts();
    }
  } catch (error) {
    console.error('获取日志统计信息失败', error);
  }
};

// 获取日志列表
const fetchLogs = async (page = pagination.current) => {
  loading.value = true;
  try {
    const [startTime, endTime] = timeRange.value;
    
    const params = {
      page,
      page_size: pagination.pageSize,
      ...filterForm,
      log_level: filterForm.log_level.join(','),
      start_time: startTime?.toISOString(),
      end_time: endTime?.toISOString(),
    };
    
    const response = await request.get('/logs', { params });
    
    if (response.data.success) {
      logs.value = response.data.data.items;
      pagination.total = response.data.data.total;
      pagination.current = page;
    }
  } catch (error) {
    console.error('获取日志列表失败', error);
  } finally {
    loading.value = false;
  }
};

// 分页变化
const onPageChange = (page) => {
  fetchLogs(page);
};

// 每页条数变化
const onPageSizeChange = (pageSize) => {
  pagination.pageSize = pageSize;
  fetchLogs(1);
};

// 重置过滤条件
const resetFilter = () => {
  filterForm.host_id = '';
  filterForm.service_id = '';
  filterForm.log_level = [];
  filterForm.keyword = '';
  
  const now = new Date();
  const yesterday = new Date();
  yesterday.setHours(now.getHours() - 24);
  timeRange.value = [yesterday, now];
  
  fetchLogs(1);
};

// 查看日志详情
const viewLogDetail = (log) => {
  currentLog.value = log;
  logDetailVisible.value = true;
};

// 导出日志
const exportLogs = () => {
  // 实际项目中应该调用后端接口进行导出
  alert('导出功能开发中...');
};

// 根据日志级别获取颜色
const getLevelColor = (level) => {
  switch (level) {
    case 'ERROR':
      return 'red';
    case 'WARN':
      return 'orange';
    case 'INFO':
      return 'blue';
    case 'DEBUG':
      return 'green';
    case 'TRACE':
      return 'gray';
    default:
      return 'gray';
  }
};

// 初始化图表
const initCharts = () => {
  // 初始化日志级别分布图表
  initLevelChart();
  
  // 初始化每日日志数量图表
  initDailyChart();
  
  // 初始化主机日志分布图表
  initHostChart();
};

// 初始化日志级别分布图表
const initLevelChart = () => {
  if (levelChart) {
    levelChart.dispose();
  }
  
  levelChart = echarts.init(levelChartRef.value);
  
  const { level_stats } = logStats.value;
  const data = Object.entries(level_stats).map(([name, value]) => ({ name, value }));
  
  const option = {
    tooltip: {
      trigger: 'item',
      formatter: '{b}: {c} ({d}%)'
    },
    legend: {
      orient: 'vertical',
      right: 10,
      top: 'center',
      data: Object.keys(level_stats)
    },
    series: [
      {
        type: 'pie',
        radius: ['50%', '70%'],
        avoidLabelOverlap: false,
        itemStyle: {
          borderRadius: 10,
          borderColor: '#fff',
          borderWidth: 2
        },
        label: {
          show: false,
          position: 'center'
        },
        emphasis: {
          label: {
            show: true,
            fontSize: '15',
            fontWeight: 'bold'
          }
        },
        labelLine: {
          show: false
        },
        data: data.map(item => ({
          name: item.name,
          value: item.value,
          itemStyle: {
            color: getLevelColor(item.name)
          }
        }))
      }
    ]
  };
  
  levelChart.setOption(option);
};

// 初始化每日日志数量图表
const initDailyChart = () => {
  if (dailyChart) {
    dailyChart.dispose();
  }
  
  dailyChart = echarts.init(dailyChartRef.value);
  
  const { daily_stats } = logStats.value;
  
  const option = {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow'
      }
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: [
      {
        type: 'category',
        data: daily_stats.map(item => item.date),
        axisTick: {
          alignWithLabel: true
        }
      }
    ],
    yAxis: [
      {
        type: 'value'
      }
    ],
    series: [
      {
        name: '日志数量',
        type: 'bar',
        barWidth: '60%',
        data: daily_stats.map(item => ({
          value: item.count,
          itemStyle: {
            color: '#165dff'
          }
        }))
      }
    ]
  };
  
  dailyChart.setOption(option);
};

// 初始化主机日志分布图表
const initHostChart = () => {
  if (hostChart) {
    hostChart.dispose();
  }
  
  hostChart = echarts.init(hostChartRef.value);
  
  const { host_stats } = logStats.value;
  
  const option = {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow'
      }
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'value'
    },
    yAxis: {
      type: 'category',
      data: host_stats.map(item => item.hostname)
    },
    series: [
      {
        name: '日志数量',
        type: 'bar',
        data: host_stats.map(item => ({
          value: item.count,
          itemStyle: {
            color: '#722ed1'
          }
        }))
      }
    ]
  };
  
  hostChart.setOption(option);
};

// 窗口大小变化时重新调整图表大小
const handleResize = () => {
  if (levelChart) levelChart.resize();
  if (dailyChart) dailyChart.resize();
  if (hostChart) hostChart.resize();
};

// 初始化
onMounted(async () => {
  // 获取基础数据
  await Promise.all([
    fetchHosts(),
    fetchServices(),
    fetchLogLevels()
  ]);
  
  // 获取日志数据
  await fetchLogs();
  
  // 获取统计数据
  await fetchLogStats();
  
  // 监听窗口大小变化
  window.addEventListener('resize', handleResize);
});

onUnmounted(() => {
  window.removeEventListener('resize', handleResize);
  if (levelChart) levelChart.dispose();
  if (dailyChart) dailyChart.dispose();
  if (hostChart) hostChart.dispose();
});
</script>

<style scoped>
.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.stat-card {
  height: 100%;
}

pre {
  background-color: #f9f9f9;
  padding: 10px;
  border-radius: 4px;
  overflow: auto;
  max-height: 400px;
  margin: 0;
}
</style> 