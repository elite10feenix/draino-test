/*
Copyright 2018 Planet Labs Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
implied. See the License for the specific language governing permissions
and limitations under the License.
*/

package kubernetes

import (
	"strings"
	"time"

	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// NewNodeLabelFilter returns a filter that returns true if the supplied object
// is a node with all of the supplied labels.
func NewNodeLabelFilter(labels map[string]string) func(o interface{}) bool {
	return func(o interface{}) bool {
		n, ok := o.(*core.Node)
		if !ok {
			return false
		}
		for k, v := range labels {
			if value, ok := n.GetLabels()[k]; value != v || !ok {
				return false
			}
		}
		return true
	}
}

// NewNodeConditionFilter returns a filter that returns true if the supplied
// object is a node with any of the supplied node conditions.
func NewNodeConditionFilter(ct []string) func(o interface{}) bool {
	return func(o interface{}) bool {
		n, ok := o.(*core.Node)
		if !ok {
			return false
		}
		if len(ct) == 0 {
			return true
		}
		pc := ParseConditions(ct)
		for _, t := range pc {
			for _, c := range n.Status.Conditions {
				if c.Type == t.Type && c.Status == t.Status && c.LastTransitionTime.Add(t.MinimumDuration).Before(time.Now()) {
					return true
				}
			}
		}
		return false
	}
}

// ParseConditions can parse the string array of conditions to a list of
// SuppliedContion to support particular status value and duration.
func ParseConditions(conditions []string) []SuppliedCondition {
	parsed := make([]SuppliedCondition, len(conditions))
	for i, c := range conditions {
		ts := strings.SplitN(c, "=", 2)
		if len(ts) != 2 {
			// Keep backward compatibility
			ts = []string{c, "True,0s"}
		}
		sm := strings.SplitN(ts[1], ",", 2)
		duration, err := time.ParseDuration(sm[1])
		if err == nil {
			parsed[i] = SuppliedCondition{core.NodeConditionType(ts[0]), core.ConditionStatus(sm[0]), duration}
		}
	}
	return parsed
}

// NodeSchedulableFilter returns true if the supplied object is a schedulable
// node.
func NodeSchedulableFilter(o interface{}) bool {
	n, ok := o.(*core.Node)
	if !ok {
		return false
	}
	return !n.Spec.Unschedulable
}

// NodeProcessed tracks whether nodes have been processed before using a map.
type NodeProcessed map[types.UID]bool

// NewNodeProcessed returns a new node processed filter.
func NewNodeProcessed() NodeProcessed {
	return make(NodeProcessed)
}

// Filter returns true if the supplied object is a node that this filter has
// not seen before. It is not threadsafe and should always be the last filter
// applied.
func (processed NodeProcessed) Filter(o interface{}) bool {
	n, ok := o.(*core.Node)
	if !ok {
		return false
	}
	if processed[n.GetUID()] {
		return false
	}
	processed[n.GetUID()] = true
	return true
}
