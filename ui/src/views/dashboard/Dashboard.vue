<template>
  <div class="page-container">
    <a-row :gutter="16">
      <a-col :span="24">
        <a-card class="mb-4">
          <a-typography-title :heading="4">集群概览</a-typography-title>
          <a-divider />
          <a-row :gutter="16">
            <a-col :span="6">
              <a-statistic 
                title="集群总数" 
                :value="statistics.clusters" 
                animation
                show-group-separator
              >
                <template #icon>
                  <a-avatar size="small" :style="{ backgroundColor: '#165dff' }">
                    <icon-apps />
                  </a-avatar>
                </template>
              </a-statistic>
            </a-col>
            <a-col :span="6">
              <a-statistic 
                title="主机总数" 
                :value="statistics.hosts" 
                animation
                show-group-separator
              >
                <template #icon>
                  <a-avatar size="small" :style="{ backgroundColor: '#37c2ff' }">
                    <icon-computer />
                  </a-avatar>
                </template>
              </a-statistic>
            </a-col>
            <a-col :span="6">
              <a-statistic 
                title="服务总数" 
                :value="statistics.services" 
                animation
                show-group-separator
              >
                <template #icon>
                  <a-avatar size="small" :style="{ backgroundColor: '#00b42a' }">
                    <icon-code-sandbox />
                  </a-avatar>
                </template>
              </a-statistic>
            </a-col>
            <a-col :span="6">
              <a-statistic 
                title="告警总数" 
                :value="statistics.alerts" 
                animation
                show-group-separator
              >
                <template #icon>
                  <a-avatar size="small" :style="{ backgroundColor: '#f53f3f' }">
                    <icon-notification />
                  </a-avatar>
                </template>
              </a-statistic>
            </a-col>
          </a-row>
        </a-card>
      </a-col>
    </a-row>

    <a-row :gutter="16">
      <a-col :span="12">
        <a-card class="mb-4">
          <template #title>
            主机资源使用情况
            <a-tooltip content="最近30分钟内主机资源使用平均值">
              <icon-question-circle class="ml-2" />
            </a-tooltip>
          </template>
          <div ref="hostResourceChartRef" style="height: 300px;"></div>
        </a-card>
      </a-col>
      <a-col :span="12">
        <a-card class="mb-4">
          <template #title>
            告警统计
            <a-tooltip content="最近7天内不同级别的告警统计">
              <icon-question-circle class="ml-2" />
            </a-tooltip>
          </template>
          <div ref="alertChartRef" style="height: 300px;"></div>
        </a-card>
      </a-col>
    </a-row>

    <a-row :gutter="16">
      <a-col :span="24">
        <a-card class="mb-4">
          <template #title>
            <a-typography-title :heading="5" style="margin: 0">
              最近告警
              <a-link href="#/alerts">查看全部</a-link>
            </a-typography-title>
          </template>
          <a-table
            :data="alertEvents"
            :loading="loadingAlerts"
            :pagination="false"
            :bordered="false"
            stripe
          >
            <template #columns>
              <a-table-column title="级别" data-index="severity" width="90">
                <template #cell="{ record }">
                  <a-tag
                    :color="getSeverityColor(record.severity)"
                    size="small"
                  >
                    {{ record.severity }}
                  </a-tag>
                </template>
              </a-table-column>
              <a-table-column title="名称" data-index="alert_name" />
              <a-table-column title="主机" data-index="hostname" />
              <a-table-column title="服务" data-index="service_name" />
              <a-table-column title="状态" data-index="status" width="120">
                <template #cell="{ record }">
                  <a-badge :status="getStatusType(record.status)" :text="record.status" />
                </template>
              </a-table-column>
              <a-table-column title="触发时间" data-index="triggered_at" />
            </template>
          </a-table>
        </a-card>
      </a-col>
    </a-row>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onUnmounted } from 'vue';
import * as echarts from 'echarts';
import request from '@/utils/request';

// 统计数据
const statistics = reactive({
  clusters: 0,
  hosts: 0,
  services: 0,
  alerts: 0
});

// 告警事件
const alertEvents = ref([]);
const loadingAlerts = ref(false);

// 图表引用
const hostResourceChartRef = ref(null);
const alertChartRef = ref(null);
let hostResourceChart = null;
let alertChart = null;

// 获取统计数据
const fetchStatistics = async () => {
  try {
    const response = await request.get('/dashboard/statistics');
    if (response.data.success) {
      Object.assign(statistics, response.data.data);
    }
  } catch (error) {
    console.error('获取统计数据失败', error);
  }
};

// 获取告警事件
const fetchAlertEvents = async () => {
  loadingAlerts.value = true;
  try {
    const response = await request.get('/alert-events', {
      params: {
        page: 1,
        page_size: 5
      }
    });
    if (response.data.success) {
      alertEvents.value = response.data.data.items;
    }
  } catch (error) {
    console.error('获取告警事件失败', error);
  } finally {
    loadingAlerts.value = false;
  }
};

// 初始化主机资源图表
const initHostResourceChart = () => {
  if (hostResourceChart) {
    hostResourceChart.dispose();
  }
  
  const chartDom = hostResourceChartRef.value;
  hostResourceChart = echarts.init(chartDom);
  
  const option = {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow'
      }
    },
    legend: {
      data: ['CPU使用率', '内存使用率', '磁盘使用率']
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'value',
      boundaryGap: [0, 0.01],
      max: 100,
      axisLabel: {
        formatter: '{value}%'
      }
    },
    yAxis: {
      type: 'category',
      data: ['host1', 'host2', 'host3', 'host4', 'host5']
    },
    series: [
      {
        name: 'CPU使用率',
        type: 'bar',
        data: [45, 37, 58, 28, 35],
        itemStyle: {
          color: '#165dff'
        }
      },
      {
        name: '内存使用率',
        type: 'bar',
        data: [68, 43, 72, 51, 63],
        itemStyle: {
          color: '#37c2ff'
        }
      },
      {
        name: '磁盘使用率',
        type: 'bar',
        data: [52, 36, 85, 42, 71],
        itemStyle: {
          color: '#00b42a'
        }
      }
    ]
  };
  
  hostResourceChart.setOption(option);
};

// 初始化告警图表
const initAlertChart = () => {
  if (alertChart) {
    alertChart.dispose();
  }
  
  const chartDom = alertChartRef.value;
  alertChart = echarts.init(chartDom);
  
  const option = {
    tooltip: {
      trigger: 'item'
    },
    legend: {
      orient: 'horizontal',
      bottom: 'bottom'
    },
    series: [
      {
        name: '告警级别',
        type: 'pie',
        radius: ['40%', '70%'],
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
        data: [
          { value: 8, name: '严重', itemStyle: { color: '#f53f3f' } },
          { value: 15, name: '警告', itemStyle: { color: '#ff7d00' } },
          { value: 22, name: '信息', itemStyle: { color: '#168cff' } }
        ]
      }
    ]
  };
  
  alertChart.setOption(option);
};

// 获取告警级别颜色
const getSeverityColor = (severity) => {
  switch (severity) {
    case 'CRITICAL':
      return 'red';
    case 'WARNING':
      return 'orange';
    case 'INFO':
      return 'blue';
    default:
      return 'gray';
  }
};

// 获取状态类型
const getStatusType = (status) => {
  switch (status) {
    case 'OPEN':
      return 'error';
    case 'ACKNOWLEDGED':
      return 'processing';
    case 'RESOLVED':
      return 'success';
    default:
      return 'default';
  }
};

// 窗口大小变化时重新调整图表大小
const handleResize = () => {
  if (hostResourceChart) {
    hostResourceChart.resize();
  }
  if (alertChart) {
    alertChart.resize();
  }
};

onMounted(async () => {
  // 获取数据
  await fetchStatistics();
  await fetchAlertEvents();
  
  // 初始化图表
  setTimeout(() => {
    initHostResourceChart();
    initAlertChart();
  }, 0);
  
  // 监听窗口大小变化
  window.addEventListener('resize', handleResize);
});

onUnmounted(() => {
  window.removeEventListener('resize', handleResize);
  if (hostResourceChart) {
    hostResourceChart.dispose();
  }
  if (alertChart) {
    alertChart.dispose();
  }
});
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style> 