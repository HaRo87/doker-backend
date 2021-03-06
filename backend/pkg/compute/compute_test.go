package compute

import (
	"github.com/haro87/dokerb/pkg/datastore"
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

const float64CompareThreshold = 0.001

func TestExtractEstimatesFailsDueToEmptyID(t *testing.T) {
	_, err := ExtractEstimatesForTask([]datastore.Estimate{}, "")
	assert.Error(t, err)
	assert.Equal(t, "Task ID cannot be empty", err.Error())
}

func TestExtractEstimatesFailsDueToEmptyEstimateList(t *testing.T) {
	_, err := ExtractEstimatesForTask([]datastore.Estimate{}, "TEST01")
	assert.Error(t, err)
	assert.Equal(t, "Not enough data to process", err.Error())
}

func TestExtractEstimatesFailsDueToEstimateForIDNotInList(t *testing.T) {
	ests := []datastore.Estimate{
		{
			TaskID: "TEST01",
		},
		{
			TaskID: "TEST02",
		},
	}

	_, err := ExtractEstimatesForTask(ests, "TEST03")
	assert.Error(t, err)
	assert.Equal(t, "Specified task with ID: TEST03 is not part of estimates", err.Error())
}

func TestExtractEstimatesSuccess(t *testing.T) {
	ests := []datastore.Estimate{
		{
			TaskID:   "TEST01",
			UserName: "Tigger",
		},
		{
			TaskID:   "TEST02",
			UserName: "Tigger",
		},
		{
			TaskID:   "TEST01",
			UserName: "Rabbit",
		},
	}

	res, err := ExtractEstimatesForTask(ests, "TEST01")
	assert.NoError(t, err)
	assert.Equal(t, "TEST01", res[0].TaskID)
	assert.Equal(t, "TEST01", res[1].TaskID)
	assert.Equal(t, "Tigger", res[0].UserName)
	assert.Equal(t, "Rabbit", res[1].UserName)
}

func TestCalculateAverageEstimateFailsDueToEmptyID(t *testing.T) {
	_, err := CalculateAverageEstimate([]datastore.Estimate{}, "")
	assert.Error(t, err)
	assert.Equal(t, "Task ID cannot be empty", err.Error())
}

func TestCalculateAverageSuccess(t *testing.T) {
	ests := []datastore.Estimate{
		{
			TaskID:         "TEST01",
			UserName:       "Tigger",
			BestCase:       1.0,
			MostLikelyCase: 2.0,
			WorstCase:      4.0,
		},
		{
			TaskID:   "TEST02",
			UserName: "Tigger",
		},
		{
			TaskID:         "TEST01",
			UserName:       "Rabbit",
			BestCase:       2.0,
			MostLikelyCase: 3.0,
			WorstCase:      5.0,
		},
	}

	res, err := CalculateAverageEstimate(ests, "TEST01")
	assert.NoError(t, err)
	assert.True(t, math.Abs(2.666-res.GetEffort()) <= float64CompareThreshold)
	assert.True(t, math.Abs(0.5-res.GetStandardDeviation()) <= float64CompareThreshold)
}

func TestGetUsersWithMaxDistanceBetweenEffortFailsDueToEmptyID(t *testing.T) {
	_, err := GetUsersWithMaxDistanceBetweenEffort([]datastore.Estimate{}, "")
	assert.Error(t, err)
	assert.Equal(t, "Task ID cannot be empty", err.Error())
}

func TestGetUsersWithMaxDistanceBetweenEffortFailsDueToWrongEffortValues(t *testing.T) {
	ests := []datastore.Estimate{
		{
			TaskID:         "TEST01",
			UserName:       "Tigger",
			BestCase:       1.0,
			MostLikelyCase: 0.2,
			WorstCase:      4.0,
		},
	}
	_, err := GetUsersWithMaxDistanceBetweenEffort(ests, "TEST01")
	assert.Error(t, err)
	assert.Equal(t, "Most Likely was smaller than Best Effort", err.Error())
}

func TestGetUsersWithMaxDistanceBetweenEffortSuccess(t *testing.T) {
	ests := []datastore.Estimate{
		{
			TaskID:         "TEST01",
			UserName:       "Tigger",
			BestCase:       1.0,
			MostLikelyCase: 2.0,
			WorstCase:      4.0,
		},
		{
			TaskID:   "TEST02",
			UserName: "Tigger",
		},
		{
			TaskID:         "TEST01",
			UserName:       "Rabbit",
			BestCase:       2.0,
			MostLikelyCase: 3.0,
			WorstCase:      5.0,
		},
		{
			TaskID:         "TEST01",
			UserName:       "Piglet",
			BestCase:       0.4,
			MostLikelyCase: 1.0,
			WorstCase:      1.2,
		},
	}

	res, err := GetUsersWithMaxDistanceBetweenEffort(ests, "TEST01")
	assert.NoError(t, err)
	assert.Equal(t, "Rabbit", res[0])
	assert.Equal(t, "Piglet", res[1])
}
