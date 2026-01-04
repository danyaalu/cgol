package combinatorics

import (
	"math/big"
)

// Binomial calculates nCk (n choose k).
func Binomial(n, k int64) *big.Int {
	if k < 0 || k > n {
		return big.NewInt(0)
	}
	if k == 0 || k == n {
		return big.NewInt(1)
	}
	if k > n/2 {
		k = n - k
	}

	res := big.NewInt(1)
	for i := int64(1); i <= k; i++ {
		res.Mul(res, big.NewInt(n-i+1))
		res.Div(res, big.NewInt(i))
	}
	return res
}

// IndexToCombination returns the k positions (0..n-1) for the given index.
// Uses the combinatorial number system (combinadics).
// The index should be in range [0, nCk - 1].
func IndexToCombination(index *big.Int, n, k int) []int {
	result := make([]int, k)
	idx := new(big.Int).Set(index)
	
	// We need to find c_k > ... > c_1 >= 0 such that
	// index = (c_k choose k) + ... + (c_1 choose 1)
	// The positions are then c_k, ..., c_1.
	
	currentK := k
	for i := 0; i < k; i++ {
		// Find largest c such that (c choose currentK) <= idx
		// We can search downwards from n-1
		c := n - 1
		for {
			binom := Binomial(int64(c), int64(currentK))
			if binom.Cmp(idx) <= 0 {
				result[i] = c
				idx.Sub(idx, binom)
				break
			}
			c--
		}
		currentK--
	}
	return result
}
