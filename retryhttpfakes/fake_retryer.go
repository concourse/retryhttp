// This file was generated by counterfeiter
package retryhttpfakes

import (
	"sync"

	"github.com/concourse/retryhttp"
)

type FakeRetryer struct {
	IsRetryableStub        func(err error) bool
	isRetryableMutex       sync.RWMutex
	isRetryableArgsForCall []struct {
		err error
	}
	isRetryableReturns struct {
		result1 bool
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeRetryer) IsRetryable(err error) bool {
	fake.isRetryableMutex.Lock()
	fake.isRetryableArgsForCall = append(fake.isRetryableArgsForCall, struct {
		err error
	}{err})
	fake.recordInvocation("IsRetryable", []interface{}{err})
	fake.isRetryableMutex.Unlock()
	if fake.IsRetryableStub != nil {
		return fake.IsRetryableStub(err)
	} else {
		return fake.isRetryableReturns.result1
	}
}

func (fake *FakeRetryer) IsRetryableCallCount() int {
	fake.isRetryableMutex.RLock()
	defer fake.isRetryableMutex.RUnlock()
	return len(fake.isRetryableArgsForCall)
}

func (fake *FakeRetryer) IsRetryableArgsForCall(i int) error {
	fake.isRetryableMutex.RLock()
	defer fake.isRetryableMutex.RUnlock()
	return fake.isRetryableArgsForCall[i].err
}

func (fake *FakeRetryer) IsRetryableReturns(result1 bool) {
	fake.IsRetryableStub = nil
	fake.isRetryableReturns = struct {
		result1 bool
	}{result1}
}

func (fake *FakeRetryer) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.isRetryableMutex.RLock()
	defer fake.isRetryableMutex.RUnlock()
	return fake.invocations
}

func (fake *FakeRetryer) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ retryhttp.Retryer = new(FakeRetryer)
