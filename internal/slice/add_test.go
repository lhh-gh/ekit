// Package slice 提供切片常用操作扩展
package slice

import (
	"github.com/lhh-gh/ekit/internal/errs"
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestAdd 测试切片插入功能
// 测试场景覆盖：
// - 有效插入位置（头部/中间/尾部）
// - 越界索引（负数/超过切片长度）
// - 触发自动扩容的情况
// - 错误处理验证
func TestAdd(t *testing.T) {
	testCases := []struct {
		name      string
		slice     []int // 原始切片
		addVal    int   // 插入值
		index     int   // 插入位置
		wantSlice []int // 预期结果切片
		wantErr   error // 预期错误
	}{
		{ // 头部插入
			name:      "index 0",
			slice:     []int{123, 100},
			addVal:    233,
			index:     0,
			wantSlice: []int{233, 123, 100},
		},
		{ // 中间位置插入
			name:      "index middle",
			slice:     []int{123, 124, 125},
			addVal:    233,
			index:     1,
			wantSlice: []int{123, 233, 124, 125},
		},
		{ // 正数越界
			name:    "index out of range",
			slice:   []int{123, 100},
			index:   12,
			wantErr: errs.NewErrIndexOutOfRange(2, 12),
		},
		{ // 负数越界
			name:    "index less than 0",
			slice:   []int{123, 100},
			index:   -1,
			wantErr: errs.NewErrIndexOutOfRange(2, -1),
		},
		{ // 尾部前插入（验证元素搬迁）
			name:      "index last",
			slice:     []int{123, 100, 101, 102, 102, 102},
			addVal:    233,
			index:     5,
			wantSlice: []int{123, 100, 101, 102, 102, 233, 102},
		},
		{ // 合法追加到末尾（长度正好允许）
			name:      "append on last",
			slice:     []int{123, 100, 101, 102, 102, 102},
			addVal:    233,
			index:     6,
			wantSlice: []int{123, 100, 101, 102, 102, 102, 233},
		},
		{ // 超出最大允许索引（len+1的位置）
			name:    "index out of range",
			slice:   []int{123, 100, 101, 102, 102, 102},
			addVal:  233,
			index:   7,
			wantErr: errs.NewErrIndexOutOfRange(6, 7),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 执行插入操作
			res, err := Add(tc.slice, tc.addVal, tc.index)

			// 验证错误匹配
			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}

			// 验证结果切片内容
			// 1. 长度是否+1
			// 2. 元素顺序是否正确
			// 3. 底层数组是否按要求扩展
			assert.Equal(t, tc.wantSlice, res)

			// 追加验证（针对尾部追加的特殊校验）
			if tc.index == len(tc.slice) {
				assert.Equal(t, tc.addVal, res[len(res)-1])
			}
		})
	}
}
