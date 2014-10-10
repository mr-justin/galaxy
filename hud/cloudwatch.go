package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/cloudwatch"
	"github.com/goamz/goamz/rds"
	"github.com/litl/galaxy/log"
	"github.com/litl/galaxy/stack"
	"github.com/litl/galaxy/utils"
)

type CloudwatchStat struct {
	Namespace  string
	Dimensions map[string]string
	MetricName string
	Statistic  string
	Component  string
}

func loadELBStats(auth aws.Auth, tscChan chan *TSCollection, done *sync.WaitGroup) {

	log.Debugf("Checking ELB...")
	defer func() {
		log.Debugf("Done checking ELB")
		done.Done()
	}()

	names, err := stack.ListActive()
	if err != nil {
		log.Errorf("ERROR: %s\n", err)
		return
	}

	for _, i := range names {

		rs, err := stack.ListStackResources(i)
		if err != nil {
			log.Errorf("ERROR: %s\n", err)
			continue
		}

		var elbName string
		for _, i := range rs.Resources {
			if i.Type == "AWS::ElasticLoadBalancing::LoadBalancer" {
				elbName = i.PhysicalId
			}
		}

		if elbName == "" {
			log.Debugf("Could not lookup ELB name for %s. Skipping.", i)
			continue
		}

		parts := strings.Split(i, "-")
		if len(parts) != 3 {
			log.Debugf("ELB %s. Does not appear to be related to galaxy. Skipping.", elbName)
			continue
		}

		resourceEnv := parts[1]
		if resourceEnv != env {
			log.Debugf("ELB %s is not part of %s. Skipping.", elbName, env)
			continue
		}

		stats = NewTSCollection()
		for _, metric := range []string{"RequestCount", "HTTPCode_Backend_2XX",
			"HTTPCode_Backend_3XX", "HTTPCode_Backend_4XX",
			"HTTPCode_Backend_5XX", "HTTPCode_ELB_4XX", "HTTPCode_ELB_5XX", "Latency",
			"HealthyHostCount", "UnHealthyHostCount", "BackendConnectionErrors",
			"SurgeQueueLength", "SpilloverCount"} {

			cwStat := CloudwatchStat{
				Namespace: "AWS/ELB",
				Dimensions: map[string]string{
					"LoadBalancerName": elbName,
				},
				MetricName: metric,
				Statistic:  "Sum",
				Component:  "elb",
			}

			if metric == "SurgeQueueLength" {
				cwStat.Statistic = "Maximum"
			}
			if metric == "Latency" || metric == "HealthyHostCount" || metric == "UnHealthyHostCount" {
				cwStat.Statistic = "Average"
			}

			prefix := fmt.Sprintf("%s.%s", env, statsPrefix)
			attr := map[string]interface{}{
				"env":       env,
				"provider":  "aws",
				"component": "elb",
				"namespace": "AWS/ELB",
				"elb":       elbName,
				"name":      metric,
			}
			err := cwStat.Load(prefix, stats, attr)
			if err != nil {
				log.Errorf("ERROR: %s\n", err)
				continue
			}
		}
		tscChan <- stats
	}
}

func loadRDSStats(auth aws.Auth, tscChan chan *TSCollection, done *sync.WaitGroup) {

	log.Debugf("Checking RDS...")
	defer func() {
		log.Debugf("Done checking RDS")
		done.Done()
	}()

	apps, err := serviceRegistry.ListApps()
	if err != nil {
		log.Errorf("ERROR: %s", err)
		return
	}

	dbs := []string{}
	for _, svcCfg := range apps {
		for k, v := range svcCfg.Env() {
			if k == "DATABASE_URL" && strings.HasPrefix(v, "postgres://") &&
				!utils.StringInSlice(v, dbs) {
				dbs = append(dbs, v)
			}
		}
	}

	reg, err := stack.GetAWSRegion("")
	if err != nil {
		log.Errorf("ERROR: %s", err)
		return
	}

	rdsClient, err := rds.New(auth, *reg)
	if err != nil {
		log.Errorf("ERROR: %s", err)
		return
	}

	resp, err := rdsClient.DescribeDBInstances("", 0, "")
	if err != nil {
		log.Errorf("ERROR: %s", err)
		return
	}

	instanceIds := []string{}
	for _, inst := range resp.DBInstances {
		addr := fmt.Sprintf("%s:%d", inst.Endpoint.Address, inst.Endpoint.Port)

		for _, url := range dbs {
			if strings.Contains(url, addr) {
				instanceIds = append(instanceIds, inst.DBInstanceIdentifier)
			}
		}
	}

	for _, dbInstance := range instanceIds {
		stats = NewTSCollection()
		for _, metric := range []string{"BinLogDiskUsage", "CPUUtilization",
			"DatabaseConnections", "DiskQueueDepth", "FreeableMemory",
			"FreeStorageSpace", "ReplicaLag", "SwapUsage", "ReadIOPS",
			"WriteIOPS", "ReadLatency", "WriteLatency", "ReadThroughput",
			"WriteThroughput", "NetworkReceiveThroughput", "NetworkTransmitThroughput"} {

			cwStat := CloudwatchStat{
				Namespace: "AWS/RDS",
				Dimensions: map[string]string{
					"DBInstanceIdentifier": dbInstance,
				},
				MetricName: metric,
				Statistic:  "Average",
				Component:  "rds",
			}

			prefix := fmt.Sprintf("%s.%s", env, statsPrefix)
			attr := map[string]interface{}{
				"env":       env,
				"provider":  "aws",
				"component": "rds",
				"namespace": "AWS/RDS",
				"instance":  dbInstance,
				"name":      metric,
			}
			err := cwStat.Load(prefix, stats, attr)
			if err != nil {
				log.Errorf("ERROR: %s\n", err)
				continue
			}
		}
		tscChan <- stats
	}
}

func loadCloudwatchStats(tscChan chan *TSCollection) {
	defer wg.Done()
	auth, err := aws.GetAuth("", "", "", time.Now().UTC())
	if err != nil {
		log.Debugf("%s. Skipping collection.", err)
		return
	}

	pollWg := sync.WaitGroup{}
	for {
		log.Debugf("Checking cloudwatch...")
		pollWg.Add(2)
		go loadELBStats(auth, tscChan, &pollWg)
		go loadRDSStats(auth, tscChan, &pollWg)
		pollWg.Wait()
		time.Sleep(60 * time.Second)
	}
}

func (c *CloudwatchStat) Load(prefix string, tsc *TSCollection, attr map[string]interface{}) error {
	ts := NewTimeSeries()

	auth, err := aws.GetAuth("", "", "", time.Now())
	if err != nil {
		return err
	}

	cw, err := cloudwatch.NewCloudWatch(auth, aws.USEast.CloudWatchServicepoint)
	if err != nil {
		return err
	}

	dimensions := []cloudwatch.Dimension{}
	for k, v := range c.Dimensions {
		dimensions = append(dimensions, cloudwatch.Dimension{
			Name:  k,
			Value: v,
		})
	}

	s, err := cw.GetMetricStatistics(&cloudwatch.GetMetricStatisticsRequest{
		Namespace:  c.Namespace,
		Dimensions: dimensions,
		MetricName: c.MetricName,
		StartTime:  time.Now().UTC().Add(-4 * time.Hour),
		EndTime:    time.Now().UTC(),
		Period:     60,
		Statistics: []string{c.Statistic},
	})
	if err != nil {
		return err
	}

	for _, metric := range s.GetMetricStatisticsResult.Datapoints {
		if c.Statistic == "Sum" {
			ts.Add(metric.Timestamp.Unix(), metric.Sum, attr)
		} else if c.Statistic == "Average" {
			ts.Add(metric.Timestamp.Unix(), metric.Average, attr)
		} else if c.Statistic == "Maximum" {
			ts.Add(metric.Timestamp.Unix(), metric.Maximum, attr)
		}
	}

	// Cloudwatch will not return a value if there is no data. Default to 0.
	unixNow := time.Now().UTC().Unix()
	secs := unixNow % 60
	if len(s.GetMetricStatisticsResult.Datapoints) == 0 {
		ts.Add(unixNow-secs, 0, attr)
	}

	key := fmt.Sprintf("%s.%s.%s", "aws", c.Component, c.MetricName)
	tsc.Get(key).AddAll(ts)
	return nil
}