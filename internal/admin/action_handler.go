package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"whistleblower_REST/internal/models"
	"whistleblower_REST/internal/notifications"
	"whistleblower_REST/internal/utils"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ActionHandler struct {
	DB *gorm.DB
}

func (h *ActionHandler) CreateAction(w http.ResponseWriter, r *http.Request) {
    reportID := chi.URLParam(r, "reportId")

    var in models.CreateActionRequest
    if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
        utils.RespondWithError(w, http.StatusBadRequest, "invalid JSON body")
        return
    }

    if in.ActionDescription == "" {
        utils.RespondWithError(w, http.StatusBadRequest, "action_description is required")
        return
    }

    // Jika HandleAt tidak diisi → gunakan waktu sekarang
    handleAt := in.HandleAt
    if handleAt == nil {
        now := time.Now()
        handleAt = &now
    }

    // Simpan ACTION
    action := models.Action{
        ID:                  uuid.NewString(),
        ActionDescription:   in.ActionDescription,
        Department:          in.Department,
        ResponsiblePerson:   in.ResponsiblePerson,
        HandleAt:            handleAt,
        EstimatedCompletion: in.EstimatedCompletion,
        ReportID:            parseUint(reportID),
        CreatedAt:           time.Now(),
        UpdatedAt:           time.Now(),
    }

    if err := h.DB.Create(&action).Error; err != nil {
        utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
        return
    }

    /*
    =====================================================
            AUTO ASSIGN INVESTIGATOR TO REPORT
    =====================================================
    ResponsiblePerson → sebenarnya investigator
    Maka kita harus update reports.investigator_id & assigned_at
    */
   if action.ResponsiblePerson != "" {
    var inv models.Admin

    // Cari investigator berdasarkan nama Responsible Person
    if err := h.DB.Where("full_name = ? AND role = ?", action.ResponsiblePerson, "investigator").
        First(&inv).Error; err == nil {

        now := time.Now()

        // Update laporan
        h.DB.Model(&models.Report{}).
            Where("id = ?", action.ReportID).
            Updates(map[string]any{
                "investigator_id": inv.ID,
                "assigned_at":     now,
            })

        fmt.Printf("[INFO] Investigator '%s' assigned to report #%d\n",
            inv.FullName, action.ReportID)
    }
}


    // Ambil status lama (untuk notifikasi perubahan status)
    var oldReport models.Report
    h.DB.First(&oldReport, action.ReportID)
    oldStatus := oldReport.Status

    // Update status laporan → menjadi ON_PROCESS
    if err := h.DB.Model(&models.Report{}).
        Where("id = ?", action.ReportID).
        Updates(map[string]any{
            "status":     models.StatusOnProcess,
            "updated_at": time.Now(),
        }).Error; err != nil {
        utils.RespondWithError(w, http.StatusInternalServerError, "failed to update report status")
        return
    }

    // Kirim notifikasi status (opsional)
    if err := notifications.NotifyStatusChange(h.DB, action.ReportID, oldStatus, models.StatusOnProcess); err != nil {
        fmt.Printf("[WARN] gagal kirim notifikasi on_process: %v\n", err)
    } else {
        fmt.Printf("[NOTIF] Notifikasi on_process terkirim untuk report #%d\n", action.ReportID)
    }

    fmt.Printf("[INFO] Report #%d marked as 'on_process' after action creation\n", action.ReportID)

    // Response sukses
    utils.RespondWithJSON(w, http.StatusOK, map[string]any{
        "message": "action created successfully",
        "action":  action,
    })
}


func (h *ActionHandler) MarkActionCompleted(w http.ResponseWriter, r *http.Request) {
    reportID := chi.URLParam(r, "reportId")

    // Cek apakah ada action
    var action models.Action
    if err := h.DB.Where("report_id = ?", reportID).First(&action).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            utils.RespondWithError(w, http.StatusNotFound, "no action found for this report")
            return
        }
        utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
        return
    }

    // Ambil report untuk cek status
    var report models.Report
    if err := h.DB.First(&report, "id = ?", reportID).Error; err != nil {
        utils.RespondWithError(w, http.StatusInternalServerError, "failed to fetch report")
        return
    }

    // 🚫 RULE: Jika laporan RESOLVED atau DISMISSED → tidak boleh diselesaikan lagi
    if report.Status == models.StatusResolved || report.Status == models.StatusDismissed {
        utils.RespondWithError(w, http.StatusForbidden,
            "cannot complete action: this report has already been closed")
        return
    }

    now := time.Now()

    // Update action completion
    if err := h.DB.Model(&models.Action{}).
        Where("report_id = ?", reportID).
        Updates(map[string]any{
            "estimated_completion": now,
            "updated_at":           now,
        }).Error; err != nil {
        utils.RespondWithError(w, http.StatusInternalServerError,
            "failed to update action completion time")
        return
    }

    // Update report → resolved
    if err := h.DB.Model(&models.Report{}).
        Where("id = ?", reportID).
        Updates(map[string]any{
            "status":     "resolved",
            "updated_at": now,
        }).Error; err != nil {
        utils.RespondWithError(w, http.StatusInternalServerError,
            "failed to update report status to resolved")
        return
    }


    // =====================================================
    // 1️⃣ Buat GLOBAL Notification
    // =====================================================
    globalNotif := models.Notification{
        Title:     "Report Resolved",
        Message:   fmt.Sprintf("Report #%d has been marked as resolved.", report.ID),
        Type:      "report_resolved",
        ReportID:  &report.ID,
        CreatedAt: now,
        IsRead:    false,
    }

    if err := h.DB.Create(&globalNotif).Error; err != nil {
        fmt.Println("[ERROR] Failed to create global notification:", err)
    }

    // =====================================================
    // 2️⃣ Buat USER Notification (jika tidak anonymous)
    // =====================================================
    if report.UserID != nil {
        userNotif := models.UserNotification{
            UserID:    *report.UserID,
            Title:     "Laporan Resolved",
            Message:   fmt.Sprintf("Laporan #%d telah selesai.", report.ID),
            Type:      "report_resolved",
            ReportID:  &report.ID,
            IsRead:    false,
            CreatedAt: now,
        }

        if err := h.DB.Create(&userNotif).Error; err != nil {
            fmt.Println("[ERROR] Failed to create user notification:", err)
        }

        // =====================================================
        // 3️⃣ Kirim PUSH NOTIF FCM ke device user
        // =====================================================

        pushData := map[string]string{
            "report_id": fmt.Sprintf("%d", report.ID),
            "type":      "report_resolved",
        }

        if err := notifications.SendPushToUser(
            h.DB,
            *report.UserID,
            "Report Resolved",
            fmt.Sprintf("Laporan Kamu #%d telah selesai.", report.ID),
            "report_resolved",
            pushData,
        ); err != nil {
            fmt.Println("[FCM] ❌ Failed to send FCM:", err)
        } else {
            fmt.Printf("[FCM] 🚀 Push sent to user %s\n", *report.UserID)
        }
    } else {
        fmt.Println("[INFO] Report is anonymous → no user notification")
    }

    fmt.Printf("[INFO] Report #%s marked as resolved\n", reportID)

    utils.RespondWithJSON(w, http.StatusOK, map[string]string{
        "message": "action completed, report resolved, notifications sent",
    })
}



// UpdateAction updates an existing action
func (h *ActionHandler) UpdateAction(w http.ResponseWriter, r *http.Request) {
	actionID := chi.URLParam(r, "actionId")
	if actionID == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "action ID is required")
		return
	}

	var req struct {
		ActionDescription   *string    `json:"action_description"`
		Department          *string    `json:"department"`
		ResponsiblePerson   *string    `json:"responsible_person"`
		HandleAt            *time.Time `json:"handle_at"`
		EstimatedCompletion *time.Time `json:"estimated_completion"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// Check if action exists
	var action models.Action
	if err := h.DB.First(&action, "id = ?", actionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.RespondWithError(w, http.StatusNotFound, "action not found")
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Build updates map
	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}

	if req.ActionDescription != nil {
		updates["action_description"] = *req.ActionDescription
	}
	if req.Department != nil {
		updates["department"] = *req.Department
	}
	if req.ResponsiblePerson != nil {
		updates["responsible_person"] = *req.ResponsiblePerson
	}
	if req.HandleAt != nil {
		updates["handle_at"] = *req.HandleAt
	}
	if req.EstimatedCompletion != nil {
		updates["estimated_completion"] = *req.EstimatedCompletion
	}

	// Check if there are any fields to update
	if len(updates) == 1 { // Only updated_at
		utils.RespondWithError(w, http.StatusBadRequest, "no fields to update")
		return
	}

	// Update action
	if err := h.DB.Model(&action).Updates(updates).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to update action")
		return
	}

	// Reload action to get updated data
	h.DB.First(&action, "id = ?", actionID)

	fmt.Printf("[INFO] Action #%s updated successfully\n", actionID)
	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message": "action updated successfully",
		"action":  action,
	})
}

func (h *ActionHandler) GetActionsByReport(w http.ResponseWriter, r *http.Request) {
	reportID := chi.URLParam(r, "reportId")

	var list []models.Action
	if err := h.DB.Where("report_id = ?", reportID).Order("created_at DESC").Find(&list).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, list)
}

func parseUint(s string) uint {
	var id uint
	fmt.Sscan(s, &id)
	return id
}
