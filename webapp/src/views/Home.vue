<template>
  <v-container>
    <v-layout row wrap style="height: 80vh;">
      <v-flex xs10 offset-xs1>

        <v-layout row wrap>
          <v-flex xs10>
            <v-text-field
              label="url"
              single-line
              outline
              v-model="url"
              @keyup.enter.native="go()"
              ></v-text-field>

          </v-flex>
          <v-flex xs2>
            <v-btn color="primary" :loading="loading" :disabled="!canGo || loading" @click="go()">
              Go
            </v-btn>
          </v-flex>
        </v-layout>
        <v-alert
          v-model="error"
          type="error"
          >
          {{ errorMsg }}
        </v-alert>
      </v-flex>

      <div id="chart"></div>
    </v-layout>
  </v-container>
</template>

<script>
// @ is an alias to /src
import Highcharts from 'highcharts';
import Vue from 'vue';
import axios from 'axios';

const locations = [
  'z0mbie42-latency-eu',
  'z0mbie42-latency-us',
];

function getURLs(url) {
  return [url, url].map((u, i) => `https://${locations[i]}.herokuapp.com/${u}`);
}

export default {
  name: 'home',
  data: () => ({
    chart: null,
    url: '',
    loading: false,
    data: null,
    error: false,
    errorMsg: '',
  }),
  computed: {
    canGo() {
      const pattern = /(http|https):\/\/(\w+:{0,1}\w*)?(\S+)(:[0-9]+)?(\/|\/([\w#!:.?+=&%!\-\/]))?/; // eslint-disable-line
      if (!pattern.test(this.url)) {
        return false;
      }

      return true;
    },
  },
  watch: {
    url(val) {
      this.$router.replace({ query: { url: val } });
    },
  },
  created() {
    this.url = this.$route.query.url;
  },
  methods: {
    async go() {
      this.error = false;
      this.errorMsg = '';
      this.loading = true;

      try {
        let data = await Promise.all(getURLs(this.url).map(u => axios.get(u)));
        data = data.map(d => d.data.data);
        console.log(data);
        Vue.nextTick(() => {
          this.renderChart(data);
          this.loading = false;
        });
      } catch (err) {
        this.error = true;
        this.errorMsg = err.toString();
      } finally {
        this.loading = false;
      }
    },
    reflowChart() {
      setTimeout(() => {
        if (this.chart) {
          this.chart.reflow();
        }
      }, 1500);
    },
    renderChart(data) {
      this.chart = Highcharts.chart('chart', {
        chart: {
          type: 'bar',
        },
        title: {
          text: 'Latency',
        },
        xAxis: {
          categories: ['eu', 'us'],
        },
        yAxis: {
          min: 0,
          title: {
            text: null,
          },
          stackLabels: {
            enabled: true,
            format: '{total} ms',
          },
        },
        legend: {
          reversed: true,
        },
        plotOptions: {
          series: {
            stacking: 'normal',
          },
          bar: {
            stacking: 'normal',
            dataLabels: {
              enabled: true,
              color: 'white',
              format: '{y} ms',
            },
          },
        },
        tooltip: {
          valueSuffix: ' ms',
        },
        series: [{
          name: 'Content Transfer',
          data: [data[0].content_transfer, data[1].content_transfer],
          color: '#b7f0f8',
        }, {
          name: 'TLS Handshake',
          data: [data[0].tls_handshake, data[1].tls_handshake],
          color: '#66d9e8',
        }, {
          name: 'Server Processing',
          data: [data[0].server_processing, data[1].server_processing],
          color: '#72c3fc',
        }, {
          name: 'TCP Connection',
          data: [data[0].tcp_connection, data[1].tcp_connection],
          color: '#91a7ff',
        }, {
          name: 'DNS Lookup',
          data: [data[0].dns_lookup, data[1].dns_lookup],
          color: '#b197fc',
        }],
      });
    },
  },
};
</script>

<style scoped>

#chart {
  width: 100%;
}

</style>
