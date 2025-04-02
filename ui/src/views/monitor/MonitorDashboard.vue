<template>
  <div class="page-container">
    <div class="filter-bar mb-4">
      <a-card>
        <a-form :model="form" layout="inline">
          <a-form-item field="host_id" label="主机">
            <a-select
              v-model="form.host_id"
              placeholder="选择主机"
              style="width: 180px"
              allow-clear
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
              v-model="form.service_id"
              placeholder="选择服务"
              style="width: 180px"
              allow-clear
            >
              <a-option
                v-for="service in services"
                :key="service.id"
                :value="service.id"
                :label="service.service_name"
              />
            </a-select>
          </a-form-item>
          <a-form-item field="metric_name" label="指标类型">
            <a-select
              v-model="form.metric_name"
              placeholder="选择指标类型"
              style="width: 180px"
              allow-clear
            >
              <a-option
                v-for="metric in metricDefinitions"
                :key="metric.metric_name"
                :value="metric.metric_name"
                :label="metric.display_name"
              />
            </a-select>
          </a-form-item>
          <a-form-item field="time_range" label="时间范围">
            <a-select
              v-model="form.time_range"
              placeholder="选择时间范围"
              style="width: 180px"
            >
              <a-option value="1h">最近1小时</a-option>
              <a-option value="6h">最近6小时</a-option>
              <a-option value="12h">最近12小时</a-option>
              <a-option value="1d">最近1天</a-option>
              <a-option value="7d">最近7天</a-option>
              <a-option value="custom">自定义</a-option>
            </a-select>
          </a-form-item>
          <a-form-item>
            <a-button type="primary" @click="loadMetricData">
              查询
            </a-button>
            <a-button style="margin-left: 8px" @click="resetFilter">
              重置
            </a-button>
          </a-form-item>
        </a-form>
      </a-card>
    </div>

    <a-row :gutter="16">
      <a-col :span="24">
        <a-card class="mb-4">
          <template #title>
            <a-space>
              <a-typography-title :heading="5" style="margin: 0">
                {{ currentMetricTitle }}
              </a-typography-title>
              <a-tag v-if="currentMetric.unit">{{ currentMetric.unit }}</a-tag>
            </a-space>
          </template>
          <div ref="chartRef" style="height: 400px;" class="metric-chart-container" />
        </a-card>
      </a-col>
    </a-row>

    <a-row :gutter="16">
      <a-col :span="24">
        <a-card class="mb-4">
          <template #title>
            <a-typography-title :heading="5" style="margin: 0">
              指标数据
            </a-typography-title>
          </template>
          <a-table
            :data="metricData"
            :loading="loading"
            :bordered="false"
            stripe
            :pagination="{
              pageSize: 10,
              showTotal: true,
              showPageSize: true
            }"
          >
            <template #columns>
              <a-table-column title="时间" data-index="timestamp" />
              <a-table-column title="主机" data-index="hostname" />
              <a-table-column title="服务" data-index="service_name" />
              <a-table-column title="数值" data-index="value">
                <template #cell="{ record }">
                  {{ formatValue(record.value) }} {{ currentMetric.unit }}
                </template>
              </a-table-column>
            </template>
          </a-table>
        </a-card>
      </a-col>
    </a-row>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onUnmounted, watch } from 'vue';
import * as echarts from 'echarts';
import request from '@/utils/request';

// 数据状态
const hosts = ref([]);
const services = ref([]);
const metricDefinitions = ref([]);
const metricData = ref([]);
const loading = ref(false);
const chartRef = ref(null);
let chart = null;

// 表单状态
const form = reactive({
  host_id: '',
  service_id: '',
  metric_name: 'cpu_usage',
  time_range: '1h'
});

// 当前指标信息
const currentMetric = reactive({
  display_name: '暂无数据',
  unit: '',
  description: ''
});

// 计算属性
const currentMetricTitle = computed(() => {
  if (!form.metric_name) return '请选择指标';
  
  const metric = metricDefinitions.value.find(m => m.metric_name === form.metric_name);
  if (metric) {
    currentMetric.display_name = metric.display_name;
    currentMetric.unit = metric.unit;
    currentMetric.description = metric.description;
    return metric.display_name;
  }
  
  return form.metric_name;
});

// 获取主机列表
const fetchHosts = async () => {
  try {
    const response = await request.get('/hosts', {
      params: {
        page_size: 100,
      }
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
      params: {
        page_size: 100,
      }
    });
    if (response.data.success) {
      services.value = response.data.data.items;
    }
  } catch (error) {
    console.error('获取服务列表失败', error);
  }
};

// 获取指标定义
const fetchMetricDefinitions = async () => {
  try {
    const response = await request.get('/metric-definitions');
    if (response.data.success) {
      metricDefinitions.value = response.data.data;
      // 设置默认显示的指标
      if (metricDefinitions.value.length > 0 && !form.metric_name) {
        form.metric_name = metricDefinitions.value[0].metric_name;
      }
    }
  } catch (error) {
    console.error('获取指标定义失败', error);
  }
};

// 加载指标数据
const loadMetricData = async () => {
  if (!form.metric_name) {
    return;
  }
  
  loading.value = true;
  
  try {
    const { startTime, endTime } = getTimeRange(form.time_range);
    
    const params = {
      metric_name: form.metric_name,
      start_time: startTime.toISOString(),
      end_time: endTime.toISOString(),
      limit: 1000,
    };
    
    if (form.host_id) {
      params.host_id = form.host_id;
    }
    
    if (form.service_id) {
      params.service_id = form.service_id;
    }
    
    const response = await request.get('/metrics', { params });
    
    if (response.data.success) {
      metricData.value = response.data.data.metrics.map(item => {
        const host = hosts.value.find(h => h.id === item.host_id);
        const service = services.value.find(s => s.id === item.service_id);
        
        return {
          ...item,
          hostname: host ? host.hostname : '',
          service_name: service ? service.service_name : '',
          timestamp: new Date(item.timestamp).toLocaleString(),
        };
      });
      
      updateChart();
    }
  } catch (error) {
    console.error('加载指标数据失败', error);
  } finally {
    loading.value = false;
  }
};

// 根据时间范围获取开始和结束时间
const getTimeRange = (range) => {
  const endTime = new Date();
  let startTime = new Date();
  
  switch (range) {
    case '1h':
      startTime.setHours(endTime.getHours() - 1);
      break;
    case '6h':
      startTime.setHours(endTime.getHours() - 6);
      break;
    case '12h':
      startTime.setHours(endTime.getHours() - 12);
      break;
    case '1d':
      startTime.setDate(endTime.getDate() - 1);
      break;
    case '7d':
      startTime.setDate(endTime.getDate() - 7);
      break;
    case 'custom':
      // 处理自定义时间范围，这里使用默认的1天
      startTime.setDate(endTime.getDate() - 1);
      break;
    default:
      startTime.setHours(endTime.getHours() - 1);
  }
  
  return { startTime, endTime };
};

// 重置过滤条件
const resetFilter = () => {
  form.host_id = '';
  form.service_id = '';
  form.time_range = '1h';
  loadMetricData();
};

// 格式化数值
const formatValue = (value) => {
  if (typeof value === 'number') {
    return value.toFixed(2);
  }
  return value;
};

// 初始化图表
const initChart = () => {
  if (chart) {
    chart.dispose();
  }
  
  chart = echarts.init(chartRef.value);
  
  // 设置默认配置
  const option = {
    tooltip: {
      trigger: 'axis',
      formatter: function(params) {
        const param = params[0];
        return `${param.name}<br />${param.seriesName}: ${formatValue(param.value)} ${currentMetric.unit}`;
      }
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: []
    },
    yAxis: {
      type: 'value',
      axisLabel: {
        formatter: `{value} ${currentMetric.unit}`
      }
    },
    series: [{
      name: '数值',
      type: 'line',
      sampling: 'average',
      itemStyle: {
        color: '#165dff'
      },
      areaStyle: {
        color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
          {
            offset: 0,
            color: 'rgba(22, 93, 255, 0.3)'
          },
          {
            offset: 1,
            color: 'rgba(22, 93, 255, 0.1)'
          }
        ])
      },
      data: []
    }]
  };
  
  chart.setOption(option);
};

// 更新图表数据
const updateChart = () => {
  if (!chart) {
    initChart();
  }
  
  const seriesData = [];
  const xAxisData = [];
  
  // 按时间排序
  const sortedData = [...metricData.value].sort((a, b) => {
    return new Date(a.timestamp) - new Date(b.timestamp);
  });
  
  sortedData.forEach(item => {
    xAxisData.push(new Date(item.timestamp).toLocaleTimeString());
    seriesData.push(parseFloat(item.value));
  });
  
  chart.setOption({
    xAxis: {
      data: xAxisData
    },
    series: [{
      name: currentMetricTitle.value,
      data: seriesData
    }],
    yAxis: {
      axisLabel: {
        formatter: `{value} ${currentMetric.unit}`
      }
    }
  });
};

// 窗口大小变化时重新调整图表大小
const handleResize = () => {
  if (chart) {
    chart.resize();
  }
};

// 监视表单变化，自动加载数据
watch(() => form.metric_name, () => {
  loadMetricData();
});

onMounted(async () => {
  await Promise.all([
    fetchHosts(),
    fetchServices(),
    fetchMetricDefinitions()
  ]);
  
  await loadMetricData();
  
  // 监听窗口大小变化
  window.addEventListener('resize', handleResize);
});

onUnmounted(() => {
  window.removeEventListener('resize', handleResize);
  if (chart) {
    chart.dispose();
  }
});
</script>

<style scoped>
.metric-chart-container {
  width: 100%;
  overflow: hidden;
}
</style> 