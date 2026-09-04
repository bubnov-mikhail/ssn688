package com.bubnov.ssn688;

import android.content.Context;
import android.util.AttributeSet;

import com.bubnov.ssn688.mobile.EbitenView;

class EbitenViewWithErrorHandling extends EbitenView {
    public EbitenViewWithErrorHandling(Context context) {
        super(context);
    }

    public EbitenViewWithErrorHandling(Context context, AttributeSet attributeSet) {
        super(context, attributeSet);
    }

    @Override
    protected void onErrorOnGameUpdate(Exception e) {
        super.onErrorOnGameUpdate(e);
    }
}
