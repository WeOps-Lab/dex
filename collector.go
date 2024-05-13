package main

import (
	"context"
	"encoding/json"
	"github.com/docker/docker/api/types/volume"
	"strings"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

var baseLabelName = []string{"container_name", "image"}
var mountLabelName = []string{"container_name", "image", "volume_name", "type", "source", "destination", "driver", "mode", "rw", "propagation"}

type DockerCollector struct {
	cli *client.Client
}

func newDockerCollector() *DockerCollector {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("can't create docker client: %v", err)
	}

	return &DockerCollector{
		cli: cli,
	}
}

func (c *DockerCollector) Describe(_ chan<- *prometheus.Desc) {

}

func (c *DockerCollector) Collect(ch chan<- prometheus.Metric) {
	containers, err := c.cli.ContainerList(context.Background(), container.ListOptions{
		All: true,
	})
	if err != nil {
		c.upMetrics(ch, float64(0))
		log.Error("can't list containers: ", err)
		return
	}

	disk, err := c.cli.DiskUsage(context.Background(), types.DiskUsageOptions{})
	if err != nil {
		c.upMetrics(ch, float64(0))
		log.Error("can't list containers: ", err)
		return
	}

	c.upMetrics(ch, float64(1))

	var wg sync.WaitGroup

	for _, eachContainer := range containers {
		wg.Add(1)
		go c.processContainer(eachContainer, ch, &wg)
	}

	for _, eachVolume := range disk.Volumes {
		wg.Add(1)
		go c.processVolume(*eachVolume, containers, ch, &wg)
	}

	wg.Wait()
}

func (c *DockerCollector) processContainer(container types.Container, ch chan<- prometheus.Metric, wg *sync.WaitGroup) {
	defer wg.Done()
	cName := strings.TrimPrefix(strings.Join(container.Names, ";"), "/")
	cImage := container.Image
	var isRunning float64
	if container.State == "running" {
		isRunning = 1
	}

	// container state metric for all containers
	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
		"container_running",
		"1 if docker container is running, 0 otherwise",
		baseLabelName,
		nil,
	), prometheus.GaugeValue, isRunning, cName, cImage)

	// stats metrics only for running containers
	if isRunning == 1 {

		if stats, err := c.cli.ContainerStats(context.Background(), container.ID, false); err != nil {
			log.Fatal(err)
		} else {
			var containerStats types.StatsJSON
			err := json.NewDecoder(stats.Body).Decode(&containerStats)
			if err != nil {
				log.Error("can't read api stats: ", err)
			}
			if err := stats.Body.Close(); err != nil {
				log.Error("can't close body: ", err)
			}

			c.blockIoMetrics(ch, &containerStats, cName, cImage)
			c.memoryMetrics(ch, &containerStats, cName, cImage)
			c.networkMetrics(ch, &container, &containerStats, cName, cImage)
			c.CPUMetrics(ch, &containerStats, cName, cImage)
			c.pidsMetrics(ch, &containerStats, cName, cImage)
		}
	}
}

func (c *DockerCollector) processVolume(volume volume.Volume, container []types.Container, ch chan<- prometheus.Metric, wg *sync.WaitGroup) {
	defer wg.Done()
	c.volumeMetrics(ch, &volume, &container)
}

func (c *DockerCollector) CPUMetrics(ch chan<- prometheus.Metric, containerStats *types.StatsJSON, cName, image string) {
	totalUsage := containerStats.CPUStats.CPUUsage.TotalUsage
	cpuDelta := totalUsage - containerStats.PreCPUStats.CPUUsage.TotalUsage
	sysemDelta := containerStats.CPUStats.SystemUsage - containerStats.PreCPUStats.SystemUsage

	cpuUtilization := float64(cpuDelta) / float64(sysemDelta) * 100.0

	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
		"cpu_utilization_percent",
		"CPU utilization in percent",
		baseLabelName,
		nil,
	), prometheus.GaugeValue, cpuUtilization, cName, image)

	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
		"cpu_utilization_seconds_total",
		"Cumulative CPU utilization in seconds",
		baseLabelName,
		nil,
	), prometheus.CounterValue, float64(totalUsage)/1e9, cName, image)
}

func (c *DockerCollector) networkMetrics(ch chan<- prometheus.Metric, container *types.Container, containerStats *types.StatsJSON, cName, image string) {
	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
		"network_rx_bytes",
		"Network received bytes total",
		baseLabelName,
		nil,
	), prometheus.CounterValue, float64(containerStats.Networks["eth0"].RxBytes), cName, image)
	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
		"network_tx_bytes",
		"Network sent bytes total",
		baseLabelName,
		nil,
	), prometheus.CounterValue, float64(containerStats.Networks["eth0"].TxBytes), cName, image)
}

func (c *DockerCollector) memoryMetrics(ch chan<- prometheus.Metric, containerStats *types.StatsJSON, cName, image string) {
	// From official documentation
	//Note: On Linux, the Docker CLI reports memory usage by subtracting page cache usage from the total memory usage.
	//The API does not perform such a calculation but rather provides the total memory usage and the amount from the page cache so that clients can use the data as needed.
	memoryUsage := containerStats.MemoryStats.Usage - containerStats.MemoryStats.Stats["cache"]
	memoryTotal := containerStats.MemoryStats.Limit

	memoryUtilization := float64(memoryUsage) / float64(memoryTotal) * 100.0
	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
		"memory_usage_bytes",
		"Total memory usage bytes",
		baseLabelName,
		nil,
	), prometheus.CounterValue, float64(memoryUsage), cName, image)
	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
		"memory_total_bytes",
		"Total memory bytes",
		baseLabelName,
		nil,
	), prometheus.CounterValue, float64(memoryTotal), cName, image)
	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
		"memory_utilization_percent",
		"Memory utilization percent",
		baseLabelName,
		nil,
	), prometheus.GaugeValue, memoryUtilization, cName, image)
}

func (c *DockerCollector) blockIoMetrics(ch chan<- prometheus.Metric, containerStats *types.StatsJSON, cName, image string) {
	var readTotal, writeTotal uint64
	for _, b := range containerStats.BlkioStats.IoServiceBytesRecursive {
		if strings.EqualFold(b.Op, "read") {
			readTotal += b.Value
		}
		if strings.EqualFold(b.Op, "write") {
			writeTotal += b.Value
		}
	}

	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
		"block_io_read_bytes",
		"Block I/O read bytes",
		baseLabelName,
		nil,
	), prometheus.CounterValue, float64(readTotal), cName, image)

	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
		"block_io_write_bytes",
		"Block I/O write bytes",
		baseLabelName,
		nil,
	), prometheus.CounterValue, float64(writeTotal), cName, image)
}

func (c *DockerCollector) pidsMetrics(ch chan<- prometheus.Metric, containerStats *types.StatsJSON, cName, image string) {
	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
		"pids_current",
		"Current number of pids in the cgroup",
		baseLabelName,
		nil,
	), prometheus.CounterValue, float64(containerStats.PidsStats.Current), cName, image)

}

func (c *DockerCollector) upMetrics(ch chan<- prometheus.Metric, up float64) {
	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
		"up",
		"docker exporter up status",
		nil,
		nil,
	), prometheus.GaugeValue, up)
}

// 卷信息
func (c *DockerCollector) volumeMetrics(ch chan<- prometheus.Metric, volume *volume.Volume, containers *[]types.Container) {
	for _, c := range *containers {
		for _, m := range c.Mounts {
			if m.Name == volume.Name {
				containerName := strings.TrimPrefix(strings.Join(c.Names, ";"), "/")
				ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
					"container_volume_usage_bytes",
					"container volume usage in bytes",
					[]string{"volume_name", "container_name"},
					nil,
				), prometheus.GaugeValue, float64(volume.UsageData.Size), volume.Name, containerName)
				// 不需要break，可以一个卷被多个容器挂载情况
			}
		}
	}

	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
		"volume_usage_bytes",
		"volume usage in bytes",
		[]string{"volume_name"},
		nil,
	), prometheus.GaugeValue, float64(volume.UsageData.Size), volume.Name)

}
