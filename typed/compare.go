/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package typed

import (
	"sigs.k8s.io/structured-merge-diff/v2/fieldpath"
	"sigs.k8s.io/structured-merge-diff/v2/schema"
	"sigs.k8s.io/structured-merge-diff/v2/value"
)

type compareWalker struct {
	lhs     value.Value
	rhs     value.Value
	schema  *schema.Schema
	typeRef schema.TypeRef

	comparison *Comparison

	inLeaf bool

	// Current path that we are merging
	path fieldpath.Path

	// Allocate only as many walkers as needed for the depth by storing them here.
	spareWalkers *[]*compareWalker
}

// merge sets w.out.
func (w *compareWalker) compare() (errs ValidationErrors) {
	if w.lhs == nil && w.rhs == nil {
		// check this condidition here instead of everywhere below.
		return errorf("at least one of lhs and rhs must be provided")
	}
	a, ok := w.schema.Resolve(w.typeRef)
	if !ok {
		return errorf("schema error: no type found matching: %v", *w.typeRef.NamedType)
	}

	alhs := deduceAtom(a, w.lhs)
	arhs := deduceAtom(a, w.rhs)
	if alhs.Equals(arhs) {
		errs = append(errs, handleAtom(arhs, w.typeRef, w)...)
	} else {
		errs = append(errs, handleAtom(alhs, w.typeRef, w)...)
		errs = append(errs, handleAtom(arhs, w.typeRef, w)...)
	}

	if !w.inLeaf {
		if w.lhs == nil {
			w.comparison.Added.Insert(w.path)
		} else if w.rhs == nil {
			w.comparison.Removed.Insert(w.path)
		}
	}

	return errs
}

// doLeaf should be called on leaves before descending into children, if there
// will be a descent. It modifies w.inLeaf.
func (w *compareWalker) doLeaf() {
	w.inLeaf = true
	// We don't recurse into leaf fields for merging.
	if w.lhs == nil {
		w.comparison.Added.Insert(w.path)
	} else if w.rhs == nil {
		w.comparison.Removed.Insert(w.path)
	} else if !value.Equals(w.rhs, w.lhs) {
		// TODO: Equality is not sufficient for this.
		// Need to implement equality check on the value type.
		w.comparison.Modified.Insert(w.path)
	}
}

func (w *compareWalker) doScalar(t *schema.Scalar) (errs ValidationErrors) {
	// All scalars are leaf fields.
	w.doLeaf()

	return nil
}

func (w *compareWalker) prepareDescent(pe fieldpath.PathElement, tr schema.TypeRef) *compareWalker {
	if w.spareWalkers == nil {
		// first descent.
		w.spareWalkers = &[]*compareWalker{}
	}
	var w2 *compareWalker
	if n := len(*w.spareWalkers); n > 0 {
		w2, *w.spareWalkers = (*w.spareWalkers)[n-1], (*w.spareWalkers)[:n-1]
	} else {
		w2 = &compareWalker{}
	}
	*w2 = *w
	w2.typeRef = tr
	w2.path = append(w2.path, pe)
	w2.lhs = nil
	w2.rhs = nil
	return w2
}

func (w *compareWalker) finishDescent(w2 *compareWalker) {
	w.path = w2.path[:len(w2.path)-1]
	*w.spareWalkers = append(*w.spareWalkers, w2)
}

func (w *compareWalker) derefMap(prefix string, v value.Value, dest *value.Map) (errs ValidationErrors) {
	// taking dest as input so that it can be called as a one-liner with
	// append.
	if v == nil {
		return nil
	}
	m, err := mapValue(v)
	if err != nil {
		return errorf("%v: %v", prefix, err)
	}
	*dest = m
	return nil
}

func (w *compareWalker) visitListItems(t *schema.List, lhs, rhs value.List) (errs ValidationErrors) {
	rLen := 0
	if rhs != nil {
		rLen = rhs.Length()
	}
	lLen := 0
	if lhs != nil {
		lLen = lhs.Length()
	}

	// This is a cheap hack to at least make the output order stable.
	rhsOrder := make([]fieldpath.PathElement, 0, rLen)

	// First, collect all RHS children.
	observedRHS := fieldpath.MakePathElementValueMap(rLen)
	if rhs != nil {
		for i := 0; i < rhs.Length(); i++ {
			child := rhs.At(i)
			pe, err := listItemToPathElement(t, i, child)
			if err != nil {
				errs = append(errs, errorf("rhs: element %v: %v", i, err.Error())...)
				// If we can't construct the path element, we can't
				// even report errors deeper in the schema, so bail on
				// this element.
				continue
			}
			if _, ok := observedRHS.Get(pe); ok {
				errs = append(errs, errorf("rhs: duplicate entries for key %v", pe.String())...)
			}
			observedRHS.Insert(pe, child)
			rhsOrder = append(rhsOrder, pe)
		}
	}

	// Then merge with LHS children.
	observedLHS := fieldpath.MakePathElementSet(lLen)
	if lhs != nil {
		for i := 0; i < lhs.Length(); i++ {
			child := lhs.At(i)
			pe, err := listItemToPathElement(t, i, child)
			if err != nil {
				errs = append(errs, errorf("lhs: element %v: %v", i, err.Error())...)
				// If we can't construct the path element, we can't
				// even report errors deeper in the schema, so bail on
				// this element.
				continue
			}
			if observedLHS.Has(pe) {
				errs = append(errs, errorf("lhs: duplicate entries for key %v", pe.String())...)
				continue
			}
			observedLHS.Insert(pe)
			w2 := w.prepareDescent(pe, t.ElementType)
			w2.lhs = value.Value(child)
			if rchild, ok := observedRHS.Get(pe); ok {
				w2.rhs = rchild
			}
			errs = append(errs, w2.compare()...)
			w.finishDescent(w2)
		}
	}

	for _, pe := range rhsOrder {
		if observedLHS.Has(pe) {
			continue
		}
		value, _ := observedRHS.Get(pe)
		w2 := w.prepareDescent(pe, t.ElementType)
		w2.rhs = value
		errs = append(errs, w2.compare()...)
		w.finishDescent(w2)
	}

	return errs
}

func (w *compareWalker) derefList(prefix string, v value.Value, dest *value.List) (errs ValidationErrors) {
	// taking dest as input so that it can be called as a one-liner with
	// append.
	if v == nil {
		return nil
	}
	l, err := listValue(v)
	if err != nil {
		return errorf("%v: %v", prefix, err)
	}
	*dest = l
	return nil
}

func (w *compareWalker) doList(t *schema.List) (errs ValidationErrors) {
	var lhs, rhs value.List
	w.derefList("lhs: ", w.lhs, &lhs)
	w.derefList("rhs: ", w.rhs, &rhs)

	// If both lhs and rhs are empty/null, treat it as a
	// leaf: this helps preserve the empty/null
	// distinction.
	emptyPromoteToLeaf := (lhs == nil || lhs.Length() == 0) && (rhs == nil || rhs.Length() == 0)

	if t.ElementRelationship == schema.Atomic || emptyPromoteToLeaf {
		w.doLeaf()
		return nil
	}
	return w.visitListItems(t, lhs, rhs)
}

func (w *compareWalker) visitMapItem(t *schema.Map, key string, lhs, rhs value.Value) (errs ValidationErrors) {
	fieldType := t.ElementType
	if sf, ok := t.FindField(key); ok {
		fieldType = sf.Type
	}
	pe := fieldpath.PathElement{FieldName: &key}
	w2 := w.prepareDescent(pe, fieldType)
	w2.lhs = lhs
	w2.rhs = rhs
	errs = append(errs, w2.compare()...)
	w.finishDescent(w2)
	return errs
}

func (w *compareWalker) visitMapItems(t *schema.Map, lhs, rhs value.Map) (errs ValidationErrors) {
	if lhs != nil {
		lhs.Iterate(func(key string, val value.Value) bool {
			var rval value.Value
			if rhs != nil {
				if item, ok := rhs.Get(key); ok {
					rval = item
					defer rval.Recycle()
				}
			}
			errs = append(errs, w.visitMapItem(t, key, val, rval)...)
			return true
		})
	}

	if rhs != nil {
		rhs.Iterate(func(key string, val value.Value) bool {
			if lhs != nil {
				if v, ok := lhs.Get(key); ok {
					v.Recycle()
					return true
				}
			}
			errs = append(errs, w.visitMapItem(t, key, nil, val)...)
			return true
		})
	}

	return errs
}

func (w *compareWalker) doMap(t *schema.Map) (errs ValidationErrors) {
	var lhs, rhs value.Map
	w.derefMap("lhs: ", w.lhs, &lhs)
	w.derefMap("rhs: ", w.rhs, &rhs)

	// If both lhs and rhs are empty/null, treat it as a
	// leaf: this helps preserve the empty/null
	// distinction.
	emptyPromoteToLeaf := (lhs == nil || lhs.Length() == 0) && (rhs == nil || rhs.Length() == 0)

	if t.ElementRelationship == schema.Atomic || emptyPromoteToLeaf {
		w.doLeaf()
		return nil
	}

	if lhs == nil && rhs == nil {
		return nil
	}

	return append(errs, w.visitMapItems(t, lhs, rhs)...)
}
