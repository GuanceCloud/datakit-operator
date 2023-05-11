package election

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTaskReassign(t *testing.T) {
	testcase := []struct {
		in  *taskManager
		out [][]string
	}{
		{
			in: &taskManager{
				tasks: []task{
					{name: "nginx", score: 8},
					{name: "redis", score: 4},
					{name: "mysql", score: 2},
					{name: "mongodb", score: 1},
					{name: "nsq", score: 1},
				},
				tasksScoreCount: 16, // 8+4+2+1+1
				workers: map[string]*worker{
					"node-1": newWorker(),
					"node-2": newWorker(),
					"node-3": newWorker(),
				},
			},
			// factor = 16/3
			out: [][]string{
				{"nginx"},            // scoreCount: 8
				{"redis", "nsq"},     // scoreCount: 5
				{"mysql", "mongodb"}, // scoreCount: 3
			},
		},

		{
			in: &taskManager{
				tasks: []task{
					{name: "nginx", score: 1},
					{name: "redis", score: 1},
					{name: "mysql", score: 1},
					{name: "mongodb", score: 1},
					{name: "kafka", score: 1},
					{name: "etcd", score: 1},
					{name: "influxdb", score: 1},
					{name: "nsq", score: 1},
				},
				tasksScoreCount: 8,
				workers: map[string]*worker{
					"node-1": newWorker(),
					"node-2": newWorker(),
					"node-3": newWorker(),
				},
			},
			// factor = 8/3
			out: [][]string{
				{"nginx", "nsq"},                      // scoreCount: 2
				{"redis", "influxdb"},                 // scoreCount: 2
				{"mysql", "etcd", "kafka", "mongodb"}, // scoreCount: 4
			},
		},

		{
			in: &taskManager{
				tasks: []task{
					{name: "nginx", score: 0},
					{name: "redis", score: 0},
					{name: "mysql", score: 0},
				},
				tasksScoreCount: 0,
				workers: map[string]*worker{
					"node-1": newWorker(),
					"node-2": newWorker(),
				},
			},
			// factor = 0/2
			out: [][]string{
				{"nginx", "mysql", "redis"}, // scoreCount: 0
			},
		},

		{
			in: &taskManager{
				tasks: []task{
					{name: "nginx", score: 2},
					{name: "redis", score: 0},
					{name: "mysql", score: 0},
				},
				tasksScoreCount: 2, // 2+0+0
				workers: map[string]*worker{
					"node-1": newWorker(),
					"node-2": newWorker(),
				},
			},
			// factor = 2/3
			out: [][]string{
				{"nginx"},          // scoreCount: 2
				{"redis", "mysql"}, // scoreCount: 0
			},
		},

		{
			in: &taskManager{
				tasks: []task{
					{name: "nginx", score: 2},
					{name: "redis", score: 1},
					{name: "mysql", score: 0},
				},
				tasksScoreCount: 3, // 2+1+0
				workers: map[string]*worker{
					"node-1": newWorker(),
					"node-2": newWorker(),
				},
			},
			// factor = 3/3
			out: [][]string{
				{"nginx"},          // scoreCount: 2
				{"redis", "mysql"}, // scoreCount: 1
			},
		},

		{
			in: &taskManager{
				tasks: []task{
					{name: "nginx", score: 4},
					{name: "redis", score: 4},
					{name: "mysql", score: 3},
					{name: "mongodb", score: 3},
					{name: "kafka", score: 2},
					{name: "etcd", score: 2},
					{name: "influxdb", score: 1},
					{name: "nsq", score: 1},
				},
				tasksScoreCount: 20, // 4+4+3+3+2+2+1+1
				workers: map[string]*worker{
					"node-1": newWorker(),
					"node-2": newWorker(),
					"node-3": newWorker(),
				},
			},
			// factor = 20/3
			out: [][]string{
				{"nginx", "nsq", "influxdb"},  // scoreCount: 6
				{"redis", "etcd"},             // scoreCount: 6
				{"mysql", "kafka", "mongodb"}, // scoreCount: 8
			},
		},

		{
			in: &taskManager{
				tasks:           []task{},
				tasksScoreCount: 0,
				workers: map[string]*worker{
					"node-1": newWorker(),
					"node-2": newWorker(),
					"node-3": newWorker(),
				},
			},
			out: nil,
		},

		{
			in: &taskManager{
				tasks: []task{
					{name: "nginx", score: 1},
					{name: "redis", score: 1},
				},
				tasksScoreCount: 2,
				workers:         map[string]*worker{},
			},
			out: nil,
		},

		{
			in: &taskManager{
				tasks: []task{
					{name: "nginx", score: 1},
					{name: "redis", score: 2},
				},
				tasksScoreCount: 3,
				workers: map[string]*worker{
					"node-1": newWorker(),
					"node-2": newWorker(),
				},
			},
			// factor = 3/2
			out: [][]string{
				{"nginx"},
				{"redis"},
			},
		},
	}

	for _, tc := range testcase {
		res := tc.in.reassignTasks()
		assert.Equal(t, tc.out, res)
	}
}
