package slice

import "github.com/lhh-gh/ekit/internal/errs"

// Add 在切片 src 的指定位置 index 处插入元素 element，并返回新切片
// 参数：
//   - src:    原始切片
//   - element: 要插入的元素
//   - index:   插入位置（0 <= index <= len(src)）
//
// 返回值：
//   - []T:    插入元素后的新切片
//   - error:  当 index 越界时返回错误
//
// 实现步骤注释：
// 1. 边界检查：确保插入位置合法（0 <= index <= len(src)）
// 2. 扩展容量：通过追加零值元素将切片长度+1
// 3. 元素后移：从最后一个元素开始向 index 后移动元素（时间复杂度O(n)）
// 4. 插入元素：将新元素放入指定位置
// 5. 返回结果：返回修改后的切片和可能的错误
//
// 注意：
//   - 当底层数组容量不足时会触发自动扩容（时间复杂度升为O(n)）
//   - 插入位置为 len(src) 时等价于追加操作
//   - 示例：
//     Add([]int{1,2}, 3, 0) => [3,1,2], nil
//     Add([]string{"a"}, "b", 5) => nil, index error
func Add[T any](src []T, element T, index int) ([]T, error) {
	length := len(src)
	// 边界校验：index不能为负数或超过当前长度
	if index < 0 || index > length {
		return nil, errs.NewErrIndexOutOfRange(length, index)
	}

	// 扩展切片容量：追加一个类型零值元素
	// 这里会处理可能的扩容，并确保有足够空间存放新元素
	var zeroValue T
	src = append(src, zeroValue)

	// 元素搬迁：从后向前移动元素（避免覆盖未移动的元素）
	// 循环条件 i > index 保证只移动插入位置之后的元素
	for i := len(src) - 1; i > index; i-- {
		src[i] = src[i-1] // 将前一个元素移到当前位置
	}
	// 插入新元素到指定位置
	src[index] = element

	// 返回新切片（底层数组可能已变更）
	return src, nil
}
