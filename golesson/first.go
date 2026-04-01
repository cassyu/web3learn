package main

import (
	"fmt"
	"sort"
)

// singleNumber 返回数组中只出现一次的数字。
// 题目前提：其余数字都恰好出现两次。
func singleNumber(nums []int) int {
	counts := make(map[int]int)
	for _, n := range nums {
		counts[n]++
	}

	for k, v := range counts {
		if v == 1 {
			return k
		}
	}
	return 0
}

func singleNumber2(nums []int) int {
	counts := make(map[int]int)
	for _, n := range nums {
		counts[n]++
	}
	for k, v := range counts {
		if v == 1 {
			return k
		}
	}
	return 0
}

// isPalindrome 判断一个整数是否为回文数。
func isPalindrome(x int) bool {
	if x < 0 {
		return false
	}

	original := x
	reversed := 0
	for x > 0 {
		reversed = reversed*10 + x%10
		x /= 10
	}
	return original == reversed
}

// isValidParentheses 判断括号字符串是否有效。
// 规则：左括号必须用相同类型的右括号闭合，且顺序正确。
func isValidParentheses(s string) bool {
	stack := make([]rune, 0, len(s))
	pairs := map[rune]rune{
		')': '(',
		']': '[',
		'}': '{',
	}

	for _, ch := range s {
		switch ch {
		case '(', '[', '{':
			stack = append(stack, ch)
		case ')', ']', '}':
			if len(stack) == 0 {
				return false
			}
			top := stack[len(stack)-1]
			if top != pairs[ch] {
				return false
			}
			stack = stack[:len(stack)-1]
		default:
			return false
		}
	}

	return len(stack) == 0
}

// longestCommonPrefix 查找字符串数组中的最长公共前缀。
func longestCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}

	prefix := strs[0]
	for i := 1; i < len(strs); i++ {
		for len(prefix) > 0 && len(strs[i]) >= 0 && strs[i][:min(len(strs[i]), len(prefix))] != prefix[:min(len(strs[i]), len(prefix))] {
			prefix = prefix[:len(prefix)-1]
		}
		for len(prefix) > len(strs[i]) {
			prefix = prefix[:len(prefix)-1]
		}
		for len(prefix) > 0 && strs[i][:len(prefix)] != prefix {
			prefix = prefix[:len(prefix)-1]
		}
		if prefix == "" {
			return ""
		}
	}
	return prefix
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// plusOne 对由数字数组表示的非负整数执行 +1。
func plusOne(digits []int) []int {
	for i := len(digits) - 1; i >= 0; i-- {
		if digits[i] < 9 {
			digits[i]++
			return digits
		}
		digits[i] = 0
	}

	// 全是 9 的情况，例如 999 -> 1000
	result := make([]int, len(digits)+1)
	result[0] = 1
	return result
}

// removeDuplicates 删除有序数组中的重复项，返回去重后的新长度。
// 要求原地修改，且额外空间复杂度为 O(1)。
func removeDuplicates(nums []int) int {
	if len(nums) == 0 {
		return 0
	}

	slow := 0
	for fast := 1; fast < len(nums); fast++ {
		if nums[fast] != nums[slow] {
			slow++
			nums[slow] = nums[fast]
		}
	}
	return slow + 1
}

// mergeIntervals 合并有重叠的区间。
func mergeIntervals(intervals [][]int) [][]int {
	if len(intervals) == 0 {
		return nil
	}

	// 按区间起点排序
	sort.Slice(intervals, func(i, j int) bool {
		return intervals[i][0] < intervals[j][0]
	})

	merged := make([][]int, 0, len(intervals))
	merged = append(merged, []int{intervals[0][0], intervals[0][1]})

	for i := 1; i < len(intervals); i++ {
		last := merged[len(merged)-1]
		curr := intervals[i]

		// 有重叠：更新末尾区间右边界
		if curr[0] <= last[1] {
			if curr[1] > last[1] {
				last[1] = curr[1]
			}
			continue
		}

		// 无重叠：直接追加新区间
		merged = append(merged, []int{curr[0], curr[1]})
	}
	return merged
}

// twoSum 返回数组中两数之和等于 target 的下标。
// 题目前提：每个输入只会对应一个答案，且不能重复使用同一元素。
func twoSum(nums []int, target int) []int {
	indexByValue := make(map[int]int, len(nums))
	for i, n := range nums {
		need := target - n
		if j, ok := indexByValue[need]; ok {
			return []int{j, i}
		}
		indexByValue[n] = i
	}
	return nil
}

func main() {
	// 题目1：只出现一次的数字
	nums := []int{4, 1, 2, 1, 2}
	fmt.Printf("singleNumber(%v) = %d\n", nums, singleNumber(nums))

	// 题目2：回文数
	tests := []int{121, -121, 10, 1221}
	for _, t := range tests {
		fmt.Printf("isPalindrome(%d) = %v\n", t, isPalindrome(t))
	}

	// 题目3：有效括号
	bracketTests := []string{"()", "()[]{}", "(]", "([)]", "{[]}"}
	for _, s := range bracketTests {
		fmt.Printf("isValidParentheses(%q) = %v\n", s, isValidParentheses(s))
	}

	// 题目4：最长公共前缀
	prefixTests := [][]string{
		{"flower", "flow", "flight"},
		{"dog", "racecar", "car"},
		{"interview", "internet", "internal"},
	}
	for _, arr := range prefixTests {
		fmt.Println("arr", arr)
		fmt.Printf("longestCommonPrefix(%v) = %q\n", arr, longestCommonPrefix(arr))
	}

	// 题目5：加一
	plusOneTests := [][]int{
		{1, 2, 3},
		{4, 3, 2, 1},
		{9},
		{9, 9, 9},
	}
	for _, d := range plusOneTests {
		fmt.Printf("plusOne(%v) = %v\n", d, plusOne(append([]int(nil), d...)))
	}

	// 题目6：删除有序数组中的重复项
	dedupTests := [][]int{
		{1, 1, 2},
		{0, 0, 1, 1, 1, 2, 2, 3, 3, 4},
	}
	for _, arr := range dedupTests {
		working := append([]int(nil), arr...)
		k := removeDuplicates(working)
		fmt.Printf("removeDuplicates(%v) -> k=%d, nums[:k]=%v\n", arr, k, working[:k])
	}

	// 题目7：合并区间
	intervalTests := [][][]int{
		{{1, 3}, {2, 6}, {8, 10}, {15, 18}},
		{{1, 4}, {4, 5}},
		{{1, 4}, {0, 4}},
	}
	for _, intervals := range intervalTests {
		working := make([][]int, len(intervals))
		for i := range intervals {
			working[i] = append([]int(nil), intervals[i]...)
		}
		fmt.Printf("mergeIntervals(%v) = %v\n", intervals, mergeIntervals(working))
	}

	// 题目8：两数之和
	twoSumTests := []struct {
		nums   []int
		target int
	}{
		{[]int{2, 7, 11, 15}, 9},
		{[]int{3, 2, 4}, 6},
		{[]int{3, 3}, 6},
	}
	for _, tc := range twoSumTests {
		fmt.Printf("twoSum(%v, %d) = %v\n", tc.nums, tc.target, twoSum(tc.nums, tc.target))
	}
}
