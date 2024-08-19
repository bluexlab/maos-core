package admin_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/navyx/ai/maos/maos-core/admin"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
	"gitlab.com/navyx/ai/maos/maos-core/internal/fixture"
	"gitlab.com/navyx/ai/maos/maos-core/internal/testhelper"
)

func TestListDeploymentsWithDB(t *testing.T) {
	t.Parallel()
	logger := testhelper.Logger(t)

	t.Run("Successful listing", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()

		accessor := dbaccess.New(dbPool)

		fixture.InsertDeployment(t, ctx, dbPool, "deployment1", []string{"user1", "user2"})
		fixture.InsertDeployment(t, ctx, dbPool, "deployment2", []string{"user3", "user4"})

		request := api.AdminListDeploymentsRequestObject{}
		response, err := admin.ListDeployments(ctx, logger, accessor, request)

		assert.NoError(t, err)
		require.IsType(t, api.AdminListDeployments200JSONResponse{}, response)

		jsonResponse := response.(api.AdminListDeployments200JSONResponse)
		assert.NotNil(t, jsonResponse.Data)
		assert.Len(t, jsonResponse.Data, 2)
		assert.Equal(t, int64(2), jsonResponse.Meta.Total)

		actualNames := lo.Map(jsonResponse.Data, func(d api.Deployment, _ int) string { return d.Name })
		assert.Equal(t, []string{"deployment2", "deployment1"}, actualNames)

		for _, deployment := range jsonResponse.Data {
			assert.NotZero(t, deployment.CreatedAt)
			assert.Equal(t, "tester", deployment.CreatedBy)
			assert.Equal(t, api.DeploymentStatusDraft, deployment.Status)
		}
	})

	t.Run("Custom page and page size", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()

		accessor := dbaccess.New(dbPool)

		lo.RepeatBy(25, func(i int) *dbsqlc.Deployment {
			return fixture.InsertDeployment(t, ctx, dbPool, fmt.Sprintf("deployment-%03d", i), nil)
		})

		request := api.AdminListDeploymentsRequestObject{
			Params: api.AdminListDeploymentsParams{
				Page:     lo.ToPtr(2),
				PageSize: lo.ToPtr(10),
			},
		}
		response, err := admin.ListDeployments(ctx, logger, accessor, request)

		assert.NoError(t, err)
		require.IsType(t, api.AdminListDeployments200JSONResponse{}, response)

		jsonResponse := response.(api.AdminListDeployments200JSONResponse)
		assert.NotNil(t, jsonResponse.Data)
		assert.Len(t, jsonResponse.Data, 10)
		assert.Equal(t, int64(25), jsonResponse.Meta.Total)

		expectedNames := lo.Map(lo.Range(10), func(i int, _ int) string { return fmt.Sprintf("deployment-%03d", 24-i-10) })
		actualNames := lo.Map(jsonResponse.Data, func(d api.Deployment, _ int) string { return d.Name })
		assert.Equal(t, expectedNames, actualNames)

		for i := 0; i < len(jsonResponse.Data)-1; i++ {
			assert.True(t, jsonResponse.Data[i].Id > jsonResponse.Data[i+1].Id)
			assert.True(t, jsonResponse.Data[i].CreatedAt >= jsonResponse.Data[i+1].CreatedAt)
		}

		for _, deployment := range jsonResponse.Data {
			assert.NotEmpty(t, deployment.Id)
			assert.NotEmpty(t, deployment.Name)
			assert.NotZero(t, deployment.CreatedAt)
			assert.NotEmpty(t, deployment.CreatedBy)
			assert.NotEmpty(t, deployment.Status)
		}
	})

	t.Run("Database pool closed", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		dbPool := testhelper.TestDB(ctx, t)

		accessor := dbaccess.New(dbPool)

		fixture.InsertDeployment(t, ctx, dbPool, "deployment1", nil)
		dbPool.Close()

		request := api.AdminListDeploymentsRequestObject{}
		response, err := admin.ListDeployments(ctx, logger, accessor, request)

		assert.NoError(t, err)
		assert.IsType(t, api.AdminListDeployments500JSONResponse{}, response)
		errorResponse := response.(api.AdminListDeployments500JSONResponse)
		assert.Contains(t, errorResponse.Error, "Cannot list deployments: closed pool")
	})
}

func TestCreateDeployment(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := testhelper.Logger(t)

	t.Run("Successful creation", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		accessor := dbaccess.New(dbPool)

		request := api.AdminCreateDeploymentRequestObject{
			Body: &api.AdminCreateDeploymentJSONRequestBody{
				Name: "test-deployment",
				User: "test-user",
			},
		}

		response, err := admin.CreateDeployment(ctx, logger, accessor, request)

		assert.NoError(t, err)
		require.IsType(t, api.AdminCreateDeployment201JSONResponse{}, response)

		createdDeployment := response.(api.AdminCreateDeployment201JSONResponse)
		assert.NotEmpty(t, createdDeployment.Data.Id)
		assert.Equal(t, "test-deployment", createdDeployment.Data.Name)
		assert.Equal(t, "test-user", createdDeployment.Data.CreatedBy)
		assert.NotZero(t, createdDeployment.Data.CreatedAt)
		assert.Equal(t, api.DeploymentStatus("draft"), createdDeployment.Data.Status)
		assert.Nil(t, createdDeployment.Data.ApprovedBy)
		assert.Zero(t, createdDeployment.Data.ApprovedAt)
		assert.Nil(t, createdDeployment.Data.FinishedBy)
		assert.Zero(t, createdDeployment.Data.FinishedAt)

		// Verify the deployment was actually inserted in the database using DeploymentList
		listRequest := api.AdminListDeploymentsRequestObject{
			Params: api.AdminListDeploymentsParams{
				Page:     lo.ToPtr(1),
				PageSize: lo.ToPtr(10),
			},
		}
		listResponse, err := admin.ListDeployments(ctx, logger, accessor, listRequest)
		assert.NoError(t, err)
		require.IsType(t, api.AdminListDeployments200JSONResponse{}, listResponse)

		listJsonResponse := listResponse.(api.AdminListDeployments200JSONResponse)
		assert.NotEmpty(t, listJsonResponse.Data)

		foundDeployment, found := lo.Find(listJsonResponse.Data, func(d api.Deployment) bool {
			return d.Id == createdDeployment.Data.Id
		})

		require.True(t, found)
		require.NotNil(t, foundDeployment)
		assert.Equal(t, createdDeployment.Data.Id, foundDeployment.Id)
		assert.Equal(t, createdDeployment.Data.Name, foundDeployment.Name)
		assert.Equal(t, createdDeployment.Data.CreatedBy, foundDeployment.CreatedBy)
		assert.Equal(t, createdDeployment.Data.CreatedAt, foundDeployment.CreatedAt)
		assert.Equal(t, createdDeployment.Data.Status, foundDeployment.Status)
	})

	t.Run("Database error", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		accessor := dbaccess.New(dbPool)

		// Close the database pool to simulate a database error
		dbPool.Close()

		request := api.AdminCreateDeploymentRequestObject{
			Body: &api.AdminCreateDeploymentJSONRequestBody{
				Name: "test-deployment",
				User: "test-user",
			},
		}

		response, err := admin.CreateDeployment(ctx, logger, accessor, request)

		assert.NoError(t, err)
		require.IsType(t, api.AdminCreateDeployment500JSONResponse{}, response)

		errorResponse := response.(api.AdminCreateDeployment500JSONResponse)
		assert.Contains(t, errorResponse.Error, "Cannot create deployment")
	})
}

func TestUpdateDeployment(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("Successful update", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		accessor := dbaccess.New(dbPool)

		// First, create a deployment to update
		createdDeployment, err := accessor.Querier().DeploymentInsert(ctx, accessor.Source(), &dbsqlc.DeploymentInsertParams{
			Name:      "test-deployment",
			CreatedBy: "test-user",
		})
		require.NoError(t, err)
		require.NotNil(t, createdDeployment)

		// Update the deployment
		updateRequest := api.AdminUpdateDeploymentRequestObject{
			Id: int(createdDeployment.ID),
			Body: &api.AdminUpdateDeploymentJSONRequestBody{
				Name:      lo.ToPtr("updated-deployment"),
				Reviewers: &[]string{"reviewer1", "reviewer2"},
			},
		}
		updateResponse, err := admin.UpdateDeployment(ctx, logger, accessor, updateRequest)
		assert.NoError(t, err)
		require.IsType(t, api.AdminUpdateDeployment200JSONResponse{}, updateResponse)

		updatedDeployment := updateResponse.(api.AdminUpdateDeployment200JSONResponse)
		assert.Equal(t, "updated-deployment", updatedDeployment.Data.Name)
		assert.EqualValues(t, createdDeployment.Status, updatedDeployment.Data.Status)
		assert.Equal(t, createdDeployment.CreatedBy, updatedDeployment.Data.CreatedBy)
		assert.Equal(t, createdDeployment.CreatedAt, updatedDeployment.Data.CreatedAt)
		assert.Equal(t, createdDeployment.ApprovedBy, updatedDeployment.Data.ApprovedBy)
		assert.Equal(t, createdDeployment.ApprovedAt, updatedDeployment.Data.ApprovedAt)
		assert.Equal(t, createdDeployment.FinishedBy, updatedDeployment.Data.FinishedBy)
		assert.Equal(t, createdDeployment.FinishedAt, updatedDeployment.Data.FinishedAt)

		// Get deployment from DB and compare
		dbDeployment, err := accessor.Querier().DeploymentGetById(ctx, accessor.Source(), createdDeployment.ID)
		require.NoError(t, err)
		assert.Equal(t, "updated-deployment", dbDeployment.Name)
		assert.Equal(t, []string{"reviewer1", "reviewer2"}, dbDeployment.Reviewers)
	})

	t.Run("Update without name", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		accessor := dbaccess.New(dbPool)

		// First, create a deployment to update
		createdDeployment, err := accessor.Querier().DeploymentInsert(ctx, accessor.Source(), &dbsqlc.DeploymentInsertParams{
			Name:      "test-deployment",
			CreatedBy: "test-user",
		})
		require.NoError(t, err)
		require.NotNil(t, createdDeployment)

		// Update the deployment without name
		updateRequest := api.AdminUpdateDeploymentRequestObject{
			Id: int(createdDeployment.ID),
			Body: &api.AdminUpdateDeploymentJSONRequestBody{
				Reviewers: &[]string{"reviewer1", "reviewer2"},
			},
		}
		updateResponse, err := admin.UpdateDeployment(ctx, logger, accessor, updateRequest)
		assert.NoError(t, err)
		require.IsType(t, api.AdminUpdateDeployment200JSONResponse{}, updateResponse)

		updatedDeployment := updateResponse.(api.AdminUpdateDeployment200JSONResponse)
		assert.Equal(t, "test-deployment", updatedDeployment.Data.Name)
		assert.EqualValues(t, createdDeployment.Status, updatedDeployment.Data.Status)
		assert.Equal(t, createdDeployment.CreatedBy, updatedDeployment.Data.CreatedBy)
		assert.Equal(t, createdDeployment.CreatedAt, updatedDeployment.Data.CreatedAt)
		assert.Equal(t, createdDeployment.ApprovedBy, updatedDeployment.Data.ApprovedBy)
		assert.Equal(t, createdDeployment.ApprovedAt, updatedDeployment.Data.ApprovedAt)
		assert.Equal(t, createdDeployment.FinishedBy, updatedDeployment.Data.FinishedBy)
		assert.Equal(t, createdDeployment.FinishedAt, updatedDeployment.Data.FinishedAt)

		// Get deployment from DB and compare
		dbDeployment, err := accessor.Querier().DeploymentGetById(ctx, accessor.Source(), createdDeployment.ID)
		require.NoError(t, err)
		assert.Equal(t, "test-deployment", dbDeployment.Name)
		assert.Equal(t, []string{"reviewer1", "reviewer2"}, dbDeployment.Reviewers)
	})

	t.Run("Update without reviewers", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		accessor := dbaccess.New(dbPool)

		// First, create a deployment to update
		createdDeployment, err := accessor.Querier().DeploymentInsert(ctx, accessor.Source(), &dbsqlc.DeploymentInsertParams{
			Name:      "test-deployment",
			CreatedBy: "test-user",
			Reviewers: []string{"initial-reviewer1", "initial-reviewer2"},
		})
		require.NoError(t, err)
		require.NotNil(t, createdDeployment)

		// Update the deployment without reviewers
		updateRequest := api.AdminUpdateDeploymentRequestObject{
			Id: int(createdDeployment.ID),
			Body: &api.AdminUpdateDeploymentJSONRequestBody{
				Name: lo.ToPtr("updated-deployment"),
			},
		}
		updateResponse, err := admin.UpdateDeployment(ctx, logger, accessor, updateRequest)
		assert.NoError(t, err)
		require.IsType(t, api.AdminUpdateDeployment200JSONResponse{}, updateResponse)

		updatedDeployment := updateResponse.(api.AdminUpdateDeployment200JSONResponse)
		assert.Equal(t, "updated-deployment", updatedDeployment.Data.Name)
		assert.EqualValues(t, createdDeployment.Status, updatedDeployment.Data.Status)
		assert.Equal(t, createdDeployment.CreatedBy, updatedDeployment.Data.CreatedBy)
		assert.Equal(t, createdDeployment.CreatedAt, updatedDeployment.Data.CreatedAt)
		assert.Equal(t, createdDeployment.ApprovedBy, updatedDeployment.Data.ApprovedBy)
		assert.Equal(t, createdDeployment.ApprovedAt, updatedDeployment.Data.ApprovedAt)
		assert.Equal(t, createdDeployment.FinishedBy, updatedDeployment.Data.FinishedBy)
		assert.Equal(t, createdDeployment.FinishedAt, updatedDeployment.Data.FinishedAt)

		// Get deployment from DB and compare
		dbDeployment, err := accessor.Querier().DeploymentGetById(ctx, accessor.Source(), createdDeployment.ID)
		require.NoError(t, err)
		assert.Equal(t, "updated-deployment", dbDeployment.Name)
		assert.Equal(t, []string{"initial-reviewer1", "initial-reviewer2"}, dbDeployment.Reviewers)
	})

	t.Run("Only draft deployment can be updated", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		accessor := dbaccess.New(dbPool)

		// First, create a deployment to update
		createRequest := api.AdminCreateDeploymentRequestObject{
			Body: &api.AdminCreateDeploymentJSONRequestBody{
				Name: "test-deployment",
				User: "test-user",
			},
		}
		createResponse, err := admin.CreateDeployment(ctx, logger, accessor, createRequest)
		require.NoError(t, err)
		createdDeployment := createResponse.(api.AdminCreateDeployment201JSONResponse)

		// Submit the deployment for review to change its status
		_, err = accessor.Querier().DeploymentSubmitForReview(ctx, accessor.Source(), int64(createdDeployment.Data.Id))
		require.NoError(t, err)

		// Attempt to update the deployment
		updateRequest := api.AdminUpdateDeploymentRequestObject{
			Id: int(createdDeployment.Data.Id),
			Body: &api.AdminUpdateDeploymentJSONRequestBody{
				Name: lo.ToPtr("updated-deployment"),
			},
		}
		updateResponse, err := admin.UpdateDeployment(ctx, logger, accessor, updateRequest)
		assert.NoError(t, err)
		require.IsType(t, api.AdminUpdateDeployment404Response{}, updateResponse)

		// Get deployment from DB and compare
		dbDeployment, err := accessor.Querier().DeploymentGetById(ctx, accessor.Source(), int64(createdDeployment.Data.Id))
		require.NoError(t, err)
		assert.Equal(t, "test-deployment", dbDeployment.Name)
		assert.EqualValues(t, "reviewing", dbDeployment.Status)
	})

	t.Run("Database error", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		accessor := dbaccess.New(dbPool)

		// First, create a deployment to update
		createdDeployment, err := accessor.Querier().DeploymentInsert(ctx, accessor.Source(), &dbsqlc.DeploymentInsertParams{
			Name:      "test-deployment",
			CreatedBy: "test-user",
		})
		require.NoError(t, err)
		require.NotNil(t, createdDeployment)

		// Close the database pool to simulate a database error
		dbPool.Close()

		// Attempt to update the deployment
		updateRequest := api.AdminUpdateDeploymentRequestObject{
			Id: int(createdDeployment.ID),
			Body: &api.AdminUpdateDeploymentJSONRequestBody{
				Name: lo.ToPtr("updated-deployment"),
			},
		}
		updateResponse, err := admin.UpdateDeployment(ctx, logger, accessor, updateRequest)
		assert.NoError(t, err)
		require.IsType(t, api.AdminUpdateDeployment500JSONResponse{}, updateResponse)

		errorResponse := updateResponse.(api.AdminUpdateDeployment500JSONResponse)
		assert.Contains(t, errorResponse.Error, "Cannot update deployment")
	})
}

func TestSubmitDeployment(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	t.Run("Submit draft deployment successfully", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		accessor := dbaccess.New(dbPool)

		// Create a draft deployment
		createdDeployment, err := accessor.Querier().DeploymentInsert(ctx, accessor.Source(), &dbsqlc.DeploymentInsertParams{
			Name:      "test-deployment",
			CreatedBy: "test-user",
		})
		require.NoError(t, err)
		require.NotNil(t, createdDeployment)

		// Submit the deployment for review
		submitRequest := api.AdminSubmitDeploymentRequestObject{Id: int(createdDeployment.ID)}
		submitResponse, err := admin.SubmitDeployment(ctx, logger, accessor, submitRequest)
		assert.NoError(t, err)
		require.IsType(t, api.AdminSubmitDeployment200Response{}, submitResponse)

		// Get deployment from DB and verify status
		dbDeployment, err := accessor.Querier().DeploymentGetById(ctx, accessor.Source(), int64(createdDeployment.ID))
		require.NoError(t, err)
		assert.EqualValues(t, "reviewing", dbDeployment.Status)
	})

	t.Run("Submit non-existent deployment", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		accessor := dbaccess.New(dbPool)

		// Attempt to submit a non-existent deployment
		submitRequest := api.AdminSubmitDeploymentRequestObject{
			Id: 999999, // Non-existent ID
		}
		submitResponse, err := admin.SubmitDeployment(ctx, logger, accessor, submitRequest)
		assert.NoError(t, err)
		require.IsType(t, api.AdminSubmitDeployment404Response{}, submitResponse)
	})

	t.Run("Submit non-draft deployment", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		accessor := dbaccess.New(dbPool)

		// Create a deployment with 'reviewing' status
		createdDeployment, err := accessor.Querier().DeploymentInsert(ctx, accessor.Source(), &dbsqlc.DeploymentInsertParams{
			Name:      "test-deployment",
			CreatedBy: "test-user",
			Status:    dbsqlc.NullDeploymentStatus{DeploymentStatus: "reviewing", Valid: true},
		})
		require.NoError(t, err)
		require.NotNil(t, createdDeployment)

		// Attempt to submit the non-draft deployment
		submitRequest := api.AdminSubmitDeploymentRequestObject{Id: int(createdDeployment.ID)}
		submitResponse, err := admin.SubmitDeployment(ctx, logger, accessor, submitRequest)
		assert.NoError(t, err)
		require.IsType(t, api.AdminSubmitDeployment404Response{}, submitResponse)

		// Verify that the status hasn't changed
		dbDeployment, err := accessor.Querier().DeploymentGetById(ctx, accessor.Source(), int64(createdDeployment.ID))
		require.NoError(t, err)
		assert.EqualValues(t, "reviewing", dbDeployment.Status)
	})

	t.Run("Database error", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		accessor := dbaccess.New(dbPool)

		// Create a draft deployment
		createdDeployment, err := accessor.Querier().DeploymentInsert(ctx, accessor.Source(), &dbsqlc.DeploymentInsertParams{
			Name:      "test-deployment",
			CreatedBy: "test-user",
			Status:    dbsqlc.NullDeploymentStatus{DeploymentStatus: "draft", Valid: true},
		})
		require.NoError(t, err)
		require.NotNil(t, createdDeployment)

		// Close the database pool to simulate a database error
		dbPool.Close()

		// Attempt to submit the deployment
		submitRequest := api.AdminSubmitDeploymentRequestObject{Id: int(createdDeployment.ID)}
		submitResponse, err := admin.SubmitDeployment(ctx, logger, accessor, submitRequest)
		assert.NoError(t, err)
		require.IsType(t, api.AdminSubmitDeployment500JSONResponse{}, submitResponse)

		errorResponse := submitResponse.(api.AdminSubmitDeployment500JSONResponse)
		assert.Contains(t, errorResponse.Error, "Cannot submit deployment")
	})
}

func TestDeleteDeployment(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	t.Run("Successfully delete draft deployment", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		accessor := dbaccess.New(dbPool)

		// Create a draft deployment
		createdDeployment, err := accessor.Querier().DeploymentInsert(ctx, accessor.Source(), &dbsqlc.DeploymentInsertParams{
			Name:      "test-deployment",
			CreatedBy: "test-user",
			Status:    dbsqlc.NullDeploymentStatus{DeploymentStatus: "draft", Valid: true},
		})
		require.NoError(t, err)
		require.NotNil(t, createdDeployment)

		// Delete the deployment
		deleteRequest := api.AdminDeleteDeploymentRequestObject{Id: int(createdDeployment.ID)}
		deleteResponse, err := admin.DeleteDeployment(ctx, logger, accessor, deleteRequest)
		assert.NoError(t, err)
		require.IsType(t, api.AdminDeleteDeployment200Response{}, deleteResponse)

		// Verify that the deployment no longer exists
		_, err = accessor.Querier().DeploymentGetById(ctx, accessor.Source(), int64(createdDeployment.ID))
		assert.Error(t, err)
		assert.Equal(t, pgx.ErrNoRows, err)
	})

	t.Run("Attempt to delete non-draft deployment", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		accessor := dbaccess.New(dbPool)

		// Create a deployment with 'reviewing' status
		createdDeployment, err := accessor.Querier().DeploymentInsert(ctx, accessor.Source(), &dbsqlc.DeploymentInsertParams{
			Name:      "test-deployment",
			CreatedBy: "test-user",
			Status:    dbsqlc.NullDeploymentStatus{DeploymentStatus: "reviewing", Valid: true},
		})
		require.NoError(t, err)
		require.NotNil(t, createdDeployment)

		rows, err := accessor.Source().Query(ctx, "SELECT id,name,status FROM deployments")
		require.NoError(t, err)
		defer rows.Close()

		for rows.Next() {
			var id int64
			var name, status string
			err := rows.Scan(&id, &name, &status)
			require.NoError(t, err)
			t.Logf("Deployment: ID=%d, Name=%s, Status=%s", id, name, status)
		}
		require.NoError(t, rows.Err())

		// Attempt to delete the non-draft deployment
		deleteRequest := api.AdminDeleteDeploymentRequestObject{Id: int(createdDeployment.ID)}
		deleteResponse, err := admin.DeleteDeployment(ctx, logger, accessor, deleteRequest)
		assert.NoError(t, err)
		require.IsType(t, api.AdminDeleteDeployment404Response{}, deleteResponse)

		// Verify that the deployment still exists
		dbDeployment, err := accessor.Querier().DeploymentGetById(ctx, accessor.Source(), int64(createdDeployment.ID))
		require.NoError(t, err)
		assert.EqualValues(t, "reviewing", dbDeployment.Status)
	})

	t.Run("Attempt to delete non-existent deployment", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		accessor := dbaccess.New(dbPool)

		// Attempt to delete a non-existent deployment
		deleteRequest := api.AdminDeleteDeploymentRequestObject{Id: 9999}
		deleteResponse, err := admin.DeleteDeployment(ctx, logger, accessor, deleteRequest)
		assert.NoError(t, err)
		require.IsType(t, api.AdminDeleteDeployment404Response{}, deleteResponse)
	})

	t.Run("Database error", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		accessor := dbaccess.New(dbPool)

		// Create a draft deployment
		createdDeployment, err := accessor.Querier().DeploymentInsert(ctx, accessor.Source(), &dbsqlc.DeploymentInsertParams{
			Name:      "test-deployment",
			CreatedBy: "test-user",
			Status:    dbsqlc.NullDeploymentStatus{DeploymentStatus: "draft", Valid: true},
		})
		require.NoError(t, err)
		require.NotNil(t, createdDeployment)

		// Close the database pool to simulate a database error
		dbPool.Close()

		// Attempt to delete the deployment
		deleteRequest := api.AdminDeleteDeploymentRequestObject{Id: int(createdDeployment.ID)}
		deleteResponse, err := admin.DeleteDeployment(ctx, logger, accessor, deleteRequest)
		assert.NoError(t, err)
		require.IsType(t, api.AdminDeleteDeployment500JSONResponse{}, deleteResponse)

		errorResponse := deleteResponse.(api.AdminDeleteDeployment500JSONResponse)
		assert.Contains(t, errorResponse.Error, "Cannot delete deployment")
	})
}
