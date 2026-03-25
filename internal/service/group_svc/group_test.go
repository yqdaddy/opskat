package group_svc

import (
	"context"
	"testing"

	"github.com/opskat/opskat/internal/model/entity/group_entity"
	"github.com/opskat/opskat/internal/repository/asset_repo"
	"github.com/opskat/opskat/internal/repository/asset_repo/mock_asset_repo"
	"github.com/opskat/opskat/internal/repository/group_repo"
	"github.com/opskat/opskat/internal/repository/group_repo/mock_group_repo"

	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func setupTest(t *testing.T) (context.Context, *mock_group_repo.MockGroupRepo, *mock_asset_repo.MockAssetRepo) {
	mockCtrl := gomock.NewController(t)
	t.Cleanup(func() { mockCtrl.Finish() })
	ctx := context.Background()
	mockGroupRepo := mock_group_repo.NewMockGroupRepo(mockCtrl)
	mockAssetRepo := mock_asset_repo.NewMockAssetRepo(mockCtrl)
	group_repo.RegisterGroup(mockGroupRepo)
	asset_repo.RegisterAsset(mockAssetRepo)
	return ctx, mockGroupRepo, mockAssetRepo
}

func TestGroupSvc_Create(t *testing.T) {
	ctx, mockGroupRepo, _ := setupTest(t)

	convey.Convey("创建分组", t, func() {
		convey.Convey("合法分组创建成功，设置时间戳", func() {
			group := &group_entity.Group{Name: "生产环境"}
			mockGroupRepo.EXPECT().Create(gomock.Any(), group).Return(nil)

			err := Group().Create(ctx, group)
			assert.NoError(t, err)
			assert.Greater(t, group.Createtime, int64(0))
			assert.Greater(t, group.Updatetime, int64(0))
		})

		convey.Convey("名称为空时 Validate 拦截，不调用 repo.Create", func() {
			group := &group_entity.Group{Name: ""}

			err := Group().Create(ctx, group)
			assert.Error(t, err)
		})
	})
}

func TestGroupSvc_Update(t *testing.T) {
	ctx, mockGroupRepo, _ := setupTest(t)

	convey.Convey("更新分组", t, func() {
		convey.Convey("合法更新成功，设置 updatetime", func() {
			group := &group_entity.Group{ID: 1, Name: "测试分组"}
			mockGroupRepo.EXPECT().Update(gomock.Any(), group).Return(nil)

			err := Group().Update(ctx, group)
			assert.NoError(t, err)
			assert.Greater(t, group.Updatetime, int64(0))
		})

		convey.Convey("名称为空时 Validate 拦截，不调用 repo.Update", func() {
			group := &group_entity.Group{ID: 1, Name: ""}

			err := Group().Update(ctx, group)
			assert.Error(t, err)
		})
	})
}

func TestGroupSvc_Delete(t *testing.T) {
	ctx, mockGroupRepo, mockAssetRepo := setupTest(t)

	convey.Convey("删除分组", t, func() {
		convey.Convey("deleteAssets=false 时，资产移到未分组（MoveToGroup）", func() {
			group := &group_entity.Group{ID: 10, ParentID: 0}
			mockGroupRepo.EXPECT().Find(gomock.Any(), int64(10)).Return(group, nil)
			mockGroupRepo.EXPECT().ReparentChildren(gomock.Any(), int64(10), int64(0)).Return(nil)
			mockAssetRepo.EXPECT().MoveToGroup(gomock.Any(), int64(10), int64(0)).Return(nil)
			mockGroupRepo.EXPECT().Delete(gomock.Any(), int64(10)).Return(nil)

			err := Group().Delete(ctx, 10, false)
			assert.NoError(t, err)
		})

		convey.Convey("deleteAssets=true 时，删除分组下资产（DeleteByGroupID）", func() {
			group := &group_entity.Group{ID: 20, ParentID: 5}
			mockGroupRepo.EXPECT().Find(gomock.Any(), int64(20)).Return(group, nil)
			mockGroupRepo.EXPECT().ReparentChildren(gomock.Any(), int64(20), int64(5)).Return(nil)
			mockAssetRepo.EXPECT().DeleteByGroupID(gomock.Any(), int64(20)).Return(nil)
			mockGroupRepo.EXPECT().Delete(gomock.Any(), int64(20)).Return(nil)

			err := Group().Delete(ctx, 20, true)
			assert.NoError(t, err)
		})
	})
}

func TestGroupSvc_Get(t *testing.T) {
	ctx, mockGroupRepo, _ := setupTest(t)

	convey.Convey("获取分组", t, func() {
		convey.Convey("委托给 repo.Find，返回对应分组", func() {
			expected := &group_entity.Group{ID: 1, Name: "运维组"}
			mockGroupRepo.EXPECT().Find(gomock.Any(), int64(1)).Return(expected, nil)

			got, err := Group().Get(ctx, 1)
			assert.NoError(t, err)
			assert.Equal(t, expected.Name, got.Name)
		})
	})
}

func TestGroupSvc_List(t *testing.T) {
	ctx, mockGroupRepo, _ := setupTest(t)

	convey.Convey("列出分组", t, func() {
		convey.Convey("委托给 repo.List，返回分组列表", func() {
			expected := []*group_entity.Group{
				{ID: 1, Name: "生产环境"},
				{ID: 2, Name: "测试环境"},
			}
			mockGroupRepo.EXPECT().List(gomock.Any()).Return(expected, nil)

			got, err := Group().List(ctx)
			assert.NoError(t, err)
			assert.Len(t, got, 2)
		})
	})
}
