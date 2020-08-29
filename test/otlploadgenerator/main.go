package main

import "time"

var (
	path      = "./data.txt" // data file path
	item      = 1000         // total number of metrics / lines in output file
	metric    = "metricName" // base metricName. output file has only unique metricName with a number suffix
	gauge     = "gauge"
	counter   = "counter"
	histogram = "histogram"
	summary   = "summary"
	types     = []string{ // types of metrics generatedq
		counter,
		gauge,
		histogram,
		summary,
	}
	labels = []string{ // each metric will have from 1 to 4 sets of labels
		"label1 value1",
		"label2 value2",
		"label3 value3",
		"label4 value4",
	}
	delimeter  = ","                        // separate name, type, labels and metric value
	space      = " "                        // separate a set of label values or metric values
	valueBound = 5000                       // metric values are [0, valueBound)
	bounds     = []float64{0.01, 0.5, 0.99} // fixed quantile/buckets

	endpoint = "localhost:55680"
	waitTime = 5 * time.Second // wait time between two sends
)

func main() {
	// Writes metrics in the following format to a text file:
	// 		name, type, label1 labelvalue1 , value1 value2 value3 value4 value5
	// gauge and counter has only one value
	generateData()

	// wait for collector to start
	time.Sleep(time.Second * 10)
	createAndSendLoad()
}